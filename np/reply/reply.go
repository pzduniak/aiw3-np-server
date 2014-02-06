package reply

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"errors"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"github.com/pzduniak/logger"
	"net"
	"reflect"
)

const signature = 0xDEADC0DE
const DATA_PACKET_SIZE = 256

var name_to_id = map[string]int{
	"HelloMessage":                            1000,
	"AuthenticateWithDetailsMessage":          1002,
	"AuthenticateKickUserMessage":             1005,
	"AuthenticateExternalStatusMessage":       1006,
	"AuthenticateResultMessage":               1010,
	"AuthenticateUserGroupMessage":            1011,
	"AuthenticateValidateTicketResultMessage": 1012,
	"AuthenticateRegisterServerResultMessage": 1022,
	"StoragePublisherFileMessage":             1111,
	"StorageUserFileMessage":                  1112,
	"StorageWriteUserFileResultMessage":       1113,
	"FriendsGetProfileDataResultMessage":      1203,
	"FriendsRosterMessage":                    1211,
	"FriendsPresenceMessage":                  1212,
	"FriendsGetUserAvatarResultMessage":       1215,
	"ServersCreateSessionResultMessage":       1302,
	"ServersGetSessionsResultMessage":         1304,
	"ServersUpdateSessionResultMessage":       1306,
	"ServersDeleteSessionResultMessage":       1308,
	"CloseAppMessage":                         2001,
}

var InvalidMessage = errors.New("Invalid message")
var NoMappingFound = errors.New("No mapping found")

func Reply(conn net.Conn, id uint32, msg proto.Message) error {
	// Reflect the passed struct
	val := reflect.ValueOf(msg)

	// Make sure it's a pointer
	if val.Kind() != reflect.Ptr {
		return InvalidMessage
	}

	// Get original struct's name
	name := val.Elem().Type().Name()

	// Map struct's name to the ID
	typeID, found := name_to_id[name]
	if !found {
		return NoMappingFound
	}

	// Marshal the data
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// Log that we are sending a message
	logger.Debugf("Sent message %d (ID: %d) to %s", typeID, id, structs.StripPort(conn.RemoteAddr().String()))

	// Generate a new data buffer
	buffer := new(bytes.Buffer)

	// Write the signature
	err = binary.Write(buffer, binary.LittleEndian, uint32(signature))
	if err != nil {
		return err
	}

	// The length of the data
	err = binary.Write(buffer, binary.LittleEndian, uint32(len(data)))
	if err != nil {
		return err
	}

	// Message's type
	err = binary.Write(buffer, binary.LittleEndian, uint32(typeID))
	if err != nil {
		return err
	}

	// Message's id (used for responses)
	err = binary.Write(buffer, binary.LittleEndian, id)
	if err != nil {
		return err
	}

	// And write the data
	_, err = buffer.Write(data)
	if err != nil {
		return err
	}

	// Pass it to the connection
	// Send a single, huge TCP packet
	_, err = buffer.WriteTo(conn)
	if err != nil {
		return err
	}

	// We're fine.
	return nil
}
