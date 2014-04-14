package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/pzduniak/aiw3-np-server/np/protocol"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"net"
)

func RPCFriendsSetPresenceMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	msg := new(protocol.FriendsSetPresenceMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	for _, a := range msg.Presence {
		connection_data.PresenceData[a.GetKey()] = a.GetValue()
	}

	return nil
}
