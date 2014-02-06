package handlers

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/protocol"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"github.com/ftrvxmtrx/gravatar"
	//"github.com/pmylund/go-cache"
	//"github.com/pzduniak/logger"
	"github.com/pzduniak/utility"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net"
	"path/filepath"
	//"strconv"
	"sync"
	//"time"
)

//var avatarCache = cache.New(time.Minute*30, time.Second*30)

var mutexCreation *sync.Mutex
var downloadMutexes map[int32]*sync.Mutex
var avatarCache map[int32][]byte

func init() {
	mutexCreation = new(sync.Mutex)
	downloadMutexes = make(map[int32]*sync.Mutex)
	avatarCache = make(map[int32][]byte)
}

func RPCFriendsGetUserAvatarMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the message
	msg := new(protocol.FriendsGetUserAvatarMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	mutexCreation.Lock()
	mutex, exists := downloadMutexes[msg.GetGuid()]
	if !exists {
		downloadMutexes[msg.GetGuid()] = new(sync.Mutex)
		mutex = downloadMutexes[msg.GetGuid()]
	}
	mutexCreation.Unlock()

	mutex.Lock()
	defer mutex.Unlock()

	// Check for data in the cache
	cached, ok := avatarCache[msg.GetGuid()]
	if ok {
		//logger.Errorf("Getting %d avatar from cache", msg.GetGuid())

		return reply.Reply(conn, packet_data.Header.Id, &protocol.FriendsGetUserAvatarResultMessage{
			Result:   proto.Int32(0),
			Guid:     msg.Guid,
			FileData: cached,
		})
	}

	// Query the database for the avatar info
	var rows []*struct {
		Email string
		Type  string
		Image string
	}

	err = environment.Env.Database.Query(`
SELECT
	email,
	avatar_type as type,
	avatar_image as image
FROM 
	misago_user
WHERE
	id = ?`, int(msg.GetGuid())).Rows(&rows)

	// Query error, something's wrong!
	if err != nil {
		return err
	}

	// No user found, might be a bug or invalid client request
	if len(rows) < 1 {
		return nil
	}

	// Load file from disk
	if rows[0].Type == "upload" {
		// Generate the filepath
		filename := filepath.Join(environment.Env.Config.NP.AvatarsPath, rows[0].Image)

		//logger.Errorf("Getting %d avatar from disk; %s", msg.GetGuid(), filename)

		// Ensure the file exists
		if !utility.FileExists(filename) {
			return nil
		}

		// Read file contents
		filecontents, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		// Get the extension of the file
		ext := filepath.Ext(filename)

		// Convert from jpg to png
		if ext == ".jpg" || ext == ".jpeg" {
			rd := bytes.NewReader(filecontents)
			img, err := jpeg.Decode(rd)
			if err != nil {
				return err
			}

			buf := make([]byte, 0)
			wr := bytes.NewBuffer(buf)
			err = png.Encode(wr, img)
			if err != nil {
				return err
			}

			filecontents = wr.Bytes()
		}

		// Convert from gif to png
		if ext == ".gif" {
			rd := bytes.NewReader(filecontents)
			img, err := gif.Decode(rd)
			if err != nil {
				return err
			}

			buf := make([]byte, 0)
			wr := bytes.NewBuffer(buf)
			err = png.Encode(wr, img)
			if err != nil {
				return err
			}

			filecontents = wr.Bytes()
		}

		// Cache it
		avatarCache[msg.GetGuid()] = filecontents
		//avatarCache.Set(strconv.Itoa(int(msg.GetGuid())), filecontents, -1)

		// Return it to the client
		return reply.Reply(conn, packet_data.Header.Id, &protocol.FriendsGetUserAvatarResultMessage{
			Result:   proto.Int32(0),
			Guid:     msg.Guid,
			FileData: filecontents,
		})
	} else if rows[0].Type == "gravatar" {
		// Download the avatar
		data, err := gravatar.GetAvatar("http", gravatar.EmailHash(rows[0].Email), 96, gravatar.DefaultIdentIcon)
		if err != nil {
			return err
		}

		//logger.Errorf("Getting %d avatar from internet; %s", msg.GetGuid(), rows[0].Email)

		// Cache it
		//avatarCache.Set(strconv.Itoa(int(msg.GetGuid())), data, -1)
		avatarCache[msg.GetGuid()] = data

		// Return it to the client
		return reply.Reply(conn, packet_data.Header.Id, &protocol.FriendsGetUserAvatarResultMessage{
			Result:   proto.Int32(0),
			Guid:     msg.Guid,
			FileData: data,
		})
	}

	return nil
}
