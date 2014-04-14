package aci

// aCI3 handler code
// Not sure if 100% working, but it handles the testapp's packets fine.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/pzduniak/aiw3-np-server/environment"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"github.com/pzduniak/aiw3-np-server/utils"
	"github.com/bamiaux/iobit"
	"github.com/pzduniak/logger"
	"hash/fnv"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
)

// Parsed private key used for the RSA-OAEP decryption
var privateKey *rsa.PrivateKey

// Loads the private key, called in main.go
func LoadKey(path string) error {
	// Read the whole file into memory
	keyData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// Decode the key from .pem to the raw data, and then parse it.
	block, _ := pem.Decode(keyData)
	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	return nil
}

// Struct passed over by the message parser.
// Packaged to make the code cleaner.
type CIResult struct {
	Authorized bool
	Tokens     [][]uint64
	Message    string
	Status     int
	Macs       []Mac
	MachineID  string
}

// I'm not sure what the fields mean, the following structure
// was used in the original NPx implementation.
type Mac struct {
	Adr   []uint64
	Token []string
}

// Called by the SendRandomStringMessage handler.
func HandleCI3(
	conn net.Conn,
	connection_data *structs.ConnData,
	packet_data *structs.PacketData,
	stringParts []string,
) error {
	// Decodes the data to raw bytes
	data, err := base64.StdEncoding.DecodeString(stringParts[1])
	if err != nil {
		return err
	}

	// First 256 characters is the header
	// It contains the AES data and is encrypted using an expensive RSA algo.
	header := data[:256]

	// The rest is encrypted using AES.
	body := data[256:]

	// Decode using RSA-OAEP-SHA1, let's assume that the private key is loaded
	keyData, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, privateKey, header, nil)
	if err != nil {
		return err
	}

	// First 32 bytes is the key of the AES-256 encoded body
	key := keyData[:32]
	// Then the 16 bytes is the initialization vector
	iv := keyData[32:48]

	// Create a new block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Create a new decrypter
	decrypter := cipher.NewCBCDecrypter(block, iv)

	// One important thing: CBC requires the length to be n*key
	// Usually is ~160 bytes long
	decData := make([]byte, len(body))
	decrypter.CryptBlocks(decData, body)

	// Use the iobit BitStream library to create a new reader
	stuff := iobit.NewReader(decData)

	// First 32 bits are a signature
	signature := stuff.Uint64Le(32)

	// I should make it uint32
	if signature != uint64(0xCAFEC0DE) {
		logger.Debugf("Bad signature from %X, got %X", connection_data.Npid, signature)
		// A kick message should be sent to the server here, but I don't think that this
		// code would ever get executed.

		/*outQueue.push({
		    status: 0,
		    authorized: false,
		    message: 'Invalid CI packet received. Restart the game and try again',
		    npid: npID
		});*/
		return nil
	}

	// To keep the packet size small, the token is hashed using FNV
	hash := fnv.New32()

	// Write current token to the hasher
	_, err = hash.Write([]byte(connection_data.Token))
	if err != nil {
		logger.Warningf("error while hashing token, %s", err)
		return err
	}

	// Next 32 bit in the stream is a slightly modified token's hash
	token := stuff.Uint64Le(32)

	// .Sum32() returns the hash
	actToken := hash.Sum32()
	sentToken := uint32(token) & 0xFFFFFFFF

	// Compare the tokens
	if actToken != sentToken {
		logger.Debug("Bad auth token from %s", connection_data.Username)
		// As I wrote earlier, a kick message should be sent here, but I don't think that
		// this code would ever get executed.

		/*outQueue.push({
		    status: 0,
		    authorized: false,
		    message: 'Invalid CI packet received',
		    npid: npID
		});*/

		return nil
	}

	// Here comes the tricky part, I wonder why the packet is formatted in such way
	var packetType uint64
	result := new(CIResult)

	// If the packet type is 2, it's the end
	for packetType != 2 {
		// Each part of the body after the initial 64 bits starts with a 5-bit "type" definition
		packetType = stuff.Uint64Le(5)

		switch packetType {
		// Client ID handling
		case 0:
			handleToken(result, stuff)
		// Cheat detection handling
		case 1:
			handleDetection(result, stuff, connection_data)
		}
	}

	// There MUST be at least a single token
	if len(result.Tokens) < 1 {
		result.Authorized = false
		result.Message = "No tokens sent"
	}

	// Allocate memory for a slice that will contain formatted tokens
	tokens := make([]string, 0)

	// Format the tokens
	for _, a := range result.Tokens {
		tokens = append(tokens, string(a[0])+":"+string(a[1])+string(a[2]))
	}

	// The user is cheating, make sure we ban him.
	if len(tokens) > 0 && (connection_data.LastCI.IsZero() || result.Status > 0) {
		var rows []*struct {
			Value  string
			Pub    int
			Reason string
		}

		// IN (?) automatically expands if a slice gets passed
		err := environment.Env.Database.Query(`
SELECT
	token_value as value,
	token_type_pub as pub,
	reason as reason
FROM
	aci_bans
WHERE
	token_value IN (?)
AND
	expires < NOW()"
		`, tokens).Rows(&rows)

		if err != nil {
			return err
		}

		// If there are any rows found, the user is banned
		if len(rows) > 0 {
			result.Authorized = false
			result.Status = 41001
			result.Message = fmt.Sprintf(
				"A ban has been issued on one or more of your tokens (%X:%s). "+
					"Stated reason: %s",
				rows[0].Pub,
				strconv.Itoa(rows[0].Pub)[:2],
				rows[0].Reason,
			)
		} else {
			// Not sure why this check is here, but I'll leave it here
			if result.Status > 0 {
				// Two-week bans for cheating
				expires := time.Now().Add(time.Hour * 24 * 14)

				// Loop over the parsed tokens
				for _, token := range tokens {
					lowerPart, err := strconv.ParseInt(string(token[2]), 16, 32)
					if err != nil {
						return err
					}

					// Insert a ban into the database
					err = environment.Env.Database.Query(`
INSERT INTO 
	aci_bans(
		user_id,
		token_type,
		token_type_pub,
		token_value,
		reason,
		expires
	)
VALUES (
	?,
	?,
	?,
	?,
	?,
	?
)`,
						structs.NpidToId(connection_data.Npid),
						tokens[0],
						(lowerPart&0xFFFF)^int64(token[0]),
						result.Message,
						expires,
					).Run()

					if err != nil {
						return err
					}
				}
			}
		}
	}

	// MAC-based connections
	// Not sure why this stuff is here, but I guess it might eventually be useful
	if connection_data.LastCI.IsZero() && len(result.Macs) > 0 {
		macs := make([]string, 0)

		for _, mac := range result.Macs {
			// Generate a HEX-encoded MAC address string
			macs = append(
				macs,
				strconv.FormatUint(mac.Adr[0], 16)+"-"+
					strconv.FormatUint(mac.Adr[1], 16)+"-"+
					strconv.FormatUint(mac.Adr[2], 16)+"-"+
					strconv.FormatUint(mac.Adr[3], 16)+"-"+
					strconv.FormatUint(mac.Adr[4], 16)+"-"+
					strconv.FormatUint(mac.Adr[5], 16),
			)
		}

		// I shouldn't use Sprintf as it's pretty slow, but it's fine, because there
		// is no blocking in the code.
		logMessage := fmt.Sprintf(
			"%X:CIin (t: %s; m: %s; g: %s)",
			connection_data.Npid,
			strings.Join(tokens, ", "),
			strings.Join(macs, ", "),
			result.MachineID,
		)

		// Add a new connection
		err = environment.Env.Database.Query(`
INSERT INTO 
	aci_connections(
		user_id,
		authorized,
		status,
		log
	)
VALUES (
	?,
	?,
	?,
	?
)`,
			structs.NpidToId(connection_data.Npid),
			result.Authorized,
			result.Status,
			logMessage,
		).Run()

		if err != nil {
			return err
		}
	}

	// If the connection is unclean, don't spam the server!
	if connection_data.IsUnclean {
		return nil
	}

	// Set LastCI as no cheat was detected
	if result.Authorized {
		connection_data.LastCI = time.Now()
		return nil
	}

	// Show the further checks that we have already sent a kick message to the server
	if result.Status > 0 {
		connection_data.IsUnclean = true
	}

	// If the player is connected to a server, kick him.
	if connection_data.ServerId != 0 {
		// Tell the server to kick the player
		// TODO: fix the msg
		return utils.KickUser(connection_data.ServerId, connection_data.Npid, int64(30000))
	}

	return nil
}

func handleToken(result *CIResult, reader *iobit.Reader) {
	// first 16 bytes are the type
	tokenType := reader.Uint64Le(16)
	// then the 32 bytes are something unclear, i guess we can make it 64bit,
	// but then the "TypeRaw" line would have to be changed
	tokenNumLow := reader.Uint64Le(32)
	tokenNumHigh := reader.Uint64Le(32)
	// Somehow mix all that stuff together
	tokenTypeRaw := (tokenNumLow & 0xFFFF) ^ tokenType

	// Add data to the result token slice.
	result.Tokens = append(result.Tokens, []uint64{
		tokenType,
		tokenNumLow,
		tokenNumHigh,
		tokenTypeRaw,
	})
}

func handleDetection(result *CIResult, reader *iobit.Reader, connection_data *structs.ConnData) {
	// First 32 bits of the detection is an uint32 defining type of the detection
	detection := reader.Uint64Le(32)

	switch detection {
	// MAC token
	// No idea why the IDs are in detection
	case 0:
		// Message format:
		// A       B  B  B  B  B  B  P    P
		// x    A*[xx xx xx xx xx xx xxxx xxxx]
		//
		// A - number of MAC ids
		// B - part of the MAC (6x uint8)
		// P - some kind of 32-bit int

		numMACs := reader.Uint64Le(4)

		for i := uint64(0); i < numMACs; i++ {
			m1 := reader.Uint64Le(8)
			m2 := reader.Uint64Le(8)
			m3 := reader.Uint64Le(8)
			m4 := reader.Uint64Le(8)
			m5 := reader.Uint64Le(8)
			m6 := reader.Uint64Le(8)
			p1 := reader.Uint64Le(32)
			p2 := reader.Uint64Le(32)

			result.Macs = append(result.Macs, Mac{
				[]uint64{
					m1,
					m2,
					m3,
					m4,
					m5,
					m6,
				},
				[]string{
					strconv.FormatUint(p1, 16),
					strconv.FormatUint(p2, 16),
				},
			})
		}
	// Machine GUID
	case 1:
		// Format:
		// A B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B
		// x xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx
		// B  B  B  B  B  B  B  B  B  B  B  B  B  B  B  B
		// xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx xx
		//
		// A - did aCI3 generate a GUID?
		// B - a single byte in the GUID

		hasToken := reader.Uint64Le(1)

		if hasToken == 0 {
			return
		}

		guid := ""

		for i := 0; i < 36; i++ {
			p := reader.Uint64Le(8)
			guid += string(p)
		}

		result.MachineID = guid
	// Legacy CI
	// aCI2 detections
	case 2:
		status := reader.Uint64Le(17)

		if status != 50001 {
			result.Authorized = false
			result.Message = "Cheat detected (" + strconv.FormatUint(status, 10) + ")"
			result.Status = int(status)
		}
	// Mutants
	// I don't even know if it's implemented in the client
	case 3:
		// Packet's structure:
		// A    A    A    A    B    C
		// xxxx xxxx xxxx xxxx xxxx xxxx
		//
		// A is the text of a window (I guess?)
		// B is the status, C is the status ^0xCAFE

		// Read the first four 32bit ints
		windowTexts := []uint64{
			reader.Uint64Le(32),
			reader.Uint64Le(32),
			reader.Uint64Le(32),
			reader.Uint64Le(32),
		}

		// Read the statuses
		status := reader.Uint64Le(32)
		statusCafe := reader.Uint64Le(32)

		// Not normal!
		if status != 0 {
			logger.Debug("status " + strconv.FormatUint(status, 10) + " cafe " + strconv.FormatUint(statusCafe, 10))
		}

		// A cheat detected
		if status == 31003 || status == 31004 {
			// Verify that statusCafe is status^0xCAFE
			if statusCafe != (status ^ 0xCAFE) {
				logger.Infof("Received invalid mutant checksum from %X", connection_data.Npid)
			}

			// List the window texts / window text decimal eqs
			if status == 31003 {
				logger.Debug("31003 window texts for %X are %s, %s, %s, %s",
					connection_data.Npid,
					strconv.FormatUint(windowTexts[0], 10),
					strconv.FormatUint(windowTexts[1], 10),
					strconv.FormatUint(windowTexts[2], 10),
					strconv.FormatUint(windowTexts[3], 10),
				)
			}

			// That's a cheat detection.
			// The most hax got detected here
			result.Authorized = false
			result.Message = "Cheat detected (" + strconv.FormatUint(status, 10) + "/" + strconv.FormatUint(windowTexts[0], 10) + ")"
			result.Status = int(status)
		}
	}
}
