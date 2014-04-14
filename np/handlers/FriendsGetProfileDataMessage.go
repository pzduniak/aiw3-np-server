package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/pzduniak/aiw3-np-server/np/protocol"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"github.com/pzduniak/logger"
	"net"
)

func RPCFriendsGetProfileDataMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	msg := new(protocol.FriendsGetProfileDataMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	logger.Debug(msg.GetProfileType())

	return nil
}
