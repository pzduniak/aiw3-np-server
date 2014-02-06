package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/protocol"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/storage"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"github.com/pzduniak/logger"
	"net"
	"strconv"
	"strings"
	"time"
)

var WrongIPAddress = errors.New("Wrong IP address")

func RPCAuthenticateWithTokenMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the data
	msg := new(protocol.AuthenticateWithTokenMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// Split the token
	token := string(msg.Token)
	parts := strings.Split(token, ":")

	// uid is the first part
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}

	// Convert it to the Npid
	npid := structs.IdToNpid(id)

	// Set connection data
	connection_data.Id = id
	connection_data.Npid = npid
	connection_data.Token = token

	// Verify!
	sess_string, err := environment.Env.Redis.Get("session:" + token).Result()
	if err != nil {
		return err
	}

	// Format is: ip;rank
	sess_parts := strings.Split(sess_string, ";")

	// Verify that user has the same IP as the one used to log in
	if sess_parts[0] != structs.StripPort(conn.RemoteAddr().String()) {
		return WrongIPAddress
	}

	// Get the rank id and save it in the session
	connection_data.RankId, err = strconv.Atoi(sess_parts[1])
	if err != nil {
		return err
	}

	// Authenticate the session
	connection_data.Authenticated = true
	connection_data.Username = sess_parts[2]

	// Add connection to the storage
	storage.SetClientConnection(npid, connection_data)

	// Return a response
	err = reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateResultMessage{
		Result:       proto.Int32(0),
		Npid:         &npid,
		SessionToken: msg.Token,
	})
	if err != nil {
		return err
	}

	// No idea what this thing is supposed to do
	time.AfterFunc(time.Millisecond*900, func() {
		err = reply.Reply(conn, 0, &protocol.AuthenticateExternalStatusMessage{
			Status: proto.Int32(0),
		})

		if err != nil {
			logger.Warning(err)
		}
	})

	return nil
}
