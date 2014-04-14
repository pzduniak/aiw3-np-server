package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/pzduniak/aiw3-np-server/np/aci"
	"github.com/pzduniak/aiw3-np-server/np/protocol"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"github.com/pzduniak/logger"
	"net"
	"strings"
)

func handleClientLeft(
	conn net.Conn,
	connection_data *structs.ConnData,
	packet_data *structs.PacketData,
	stringParts []string,
) error {
	// TODO: Add it to event system
	logger.Debug(strings.Join(stringParts, " "))
	return nil
}

func handlePortAnnounced(
	conn net.Conn,
	connection_data *structs.ConnData,
	packet_data *structs.PacketData,
	stringParts []string,
) error {
	connection_data.ServerAddr = structs.StripPort(conn.RemoteAddr().String()) + ":" + stringParts[1]
	return nil
}

func RPCStorageSendRandomStringMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Parse the message
	msg := new(protocol.StorageSendRandomStringMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// Get the string and split it by spaces
	random_string := msg.GetRandomString()
	parts := strings.Fields(random_string)

	if len(parts) != 2 {
		return nil
	}

	// First part of the string is a header
	switch parts[0] {
	case "troll":
		// Deprecated
		logger.Debugf("Handling aCI2 request from %X", connection_data.Npid)
		return aci.HandleCI2(conn, connection_data, packet_data, parts)
	case "roll":
		// Will be used soon
		logger.Debugf("Handling aCI3 request from %X", connection_data.Npid)
		return aci.HandleCI3(conn, connection_data, packet_data, parts)
	case "fal":
		// Currently used
		logger.Debugf("Handling aCI2.5 request from %X", connection_data.Npid)
		return aci.HandleCI25(conn, connection_data, packet_data, parts)
	case "dis":
		// Should call the events system
		logger.Debugf("Handling disconnect message from %X", connection_data.Npid)
		return handleClientLeft(conn, connection_data, packet_data, parts)
	case "port":
		// Not sure if it is used
		logger.Debugf("Handling port announce from %X", connection_data.Npid)
		return handlePortAnnounced(conn, connection_data, packet_data, parts)
	}

	return nil
}
