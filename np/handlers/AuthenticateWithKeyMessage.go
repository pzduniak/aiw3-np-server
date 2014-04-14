package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/pzduniak/aiw3-np-server/np/protocol"
	"github.com/pzduniak/aiw3-np-server/np/reply"
	"github.com/pzduniak/aiw3-np-server/np/storage"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"net"
)

func RPCAuthenticateWithKeyMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the data
	msg := new(protocol.AuthenticateWithKeyMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// Generate a new connection-id based npid
	npid := structs.IdToNpid(connection_data.ConnectionId)

	// Fill in the connection data
	connection_data.Authenticated = true
	connection_data.IsServer = true
	connection_data.Token = ""
	connection_data.Npid = npid

	// Add connection to the storage
	storage.SetServerConnection(npid, connection_data)

	// Reply with the data
	return reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateResultMessage{
		Result:       proto.Int32(0),
		Npid:         proto.Uint64(npid),
		SessionToken: []byte(""),
	})
}
