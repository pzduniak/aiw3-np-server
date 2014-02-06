package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/storage"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/protocol/auth"
	"net"
)

func RPCAuthenticateWithKeyMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the data
	msg := new(auth.AuthenticateWithKeyMessage)
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
	return reply.Reply(conn, packet_data.Header.Id, &auth.AuthenticateResultMessage{
		Result:       proto.Int32(0),
		Npid:         proto.Uint64(npid),
		SessionToken: []byte(""),
	})
}
