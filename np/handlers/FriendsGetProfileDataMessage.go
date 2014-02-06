package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/protocol/friends"
	"github.com/pzduniak/logger"
	"net"
)

func RPCFriendsGetProfileDataMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	msg := new(friends.FriendsGetProfileDataMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	logger.Debug(msg.GetProfileType())

	return nil
}
