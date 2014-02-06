package np

import (
	"errors"
	"git.cloudrack.io/aiw3/np-server/np/handlers"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"net"
)

var NoHandlerFound = errors.New("No handler found")

func HandleMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	switch packet_data.Header.Type {
	case 1001:
		return handlers.RPCAuthenticateWithKeyMessage(conn, connection_data, packet_data)
	case 1003:
		return handlers.RPCAuthenticateWithTokenMessage(conn, connection_data, packet_data)
	case 1004:
		return handlers.RPCAuthenticateValidateTicketMessage(conn, connection_data, packet_data)
	case 1021:
		return handlers.RPCAuthenticateRegisterServerMessage(conn, connection_data, packet_data)
	case 1101:
		return handlers.RPCStorageGetPublisherFileMessage(conn, connection_data, packet_data)
	case 1102:
		return handlers.RPCStorageGetUserFileMessage(conn, connection_data, packet_data)
	case 1103:
		return handlers.RPCStorageWriteUserFileMessage(conn, connection_data, packet_data)
	case 1104:
		return handlers.RPCStorageSendRandomStringMessage(conn, connection_data, packet_data)
	case 1201:
		return handlers.RPCFriendsSetSteamIDMessage(conn, connection_data, packet_data)
	case 1202:
		return handlers.RPCFriendsGetProfileDataMessage(conn, connection_data, packet_data)
	case 1213:
		return handlers.RPCFriendsSetPresenceMessage(conn, connection_data, packet_data)
	case 1214:
		return handlers.RPCFriendsGetUserAvatarMessage(conn, connection_data, packet_data)
	case 1301:
		return handlers.RPCServersCreateSessionMessage(conn, connection_data, packet_data)
	case 1303:
		return handlers.RPCServersGetSessionsMessage(conn, connection_data, packet_data)
	case 1305:
		return handlers.RPCServersUpdateSessionMessage(conn, connection_data, packet_data)
	case 1307:
		return handlers.RPCServersDeleteSessionMessage(conn, connection_data, packet_data)
	case 2002:
		return handlers.RPCMessagingSendDataMessage(conn, connection_data, packet_data)
	}

	return NoHandlerFound
}
