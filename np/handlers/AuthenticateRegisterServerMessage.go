package handlers

import (
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"net"
)

func RPCAuthenticateRegisterServerMessage(conn net.Conn, data *structs.ConnData, packet_data *structs.PacketData) error {
	// This handler is used for server license keys. Not needed right now.
	// Maybe later?
	return nil
}
