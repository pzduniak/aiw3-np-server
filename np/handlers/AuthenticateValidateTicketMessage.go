package handlers

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"github.com/pzduniak/aiw3-np-server/np/protocol"
	"github.com/pzduniak/aiw3-np-server/np/reply"
	"github.com/pzduniak/aiw3-np-server/np/storage"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	//"github.com/pzduniak/aiw3-np-server/utils"
	//"github.com/pzduniak/logger"
	"net"
	//"time"
)

func RPCAuthenticateValidateTicketMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the data
	msg := new(protocol.AuthenticateValidateTicketMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// Create a new buffer based on the ticket data, in order to read data from it.
	buf := bytes.NewBuffer(msg.Ticket)

	// Structure:
	// <xxxx>  <xxxxxxxx> <xxxxxxxx> <xxxx>
	// version clientID   serverID   timeIssued

	var version, issued uint32
	var clientId, serverId uint64

	err = binary.Read(buf, binary.LittleEndian, &version)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &clientId)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &serverId)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &issued)
	if err != nil {
		return err
	}

	// Only version 1 is valid
	if version == 1 {
		// Verify that the request isn't spoofed
		if connection_data.Npid == serverId {
			// Get client's connection
			session := storage.GetClientConnection(clientId)

			// Make sure the session is here
			if session == nil {
				return nil
			}

			// Make sure it's not a server
			if session.IsServer {
				return nil
			}

			// Make sure that the IDs are valid
			if session.Npid != clientId {
				return reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateValidateTicketResultMessage{
					Result:  proto.Int32(1),
					Npid:    &clientId,
					GroupID: proto.Int32(1),
				})
			}

			// Make sure there's no aCI detection
			if session.IsUnclean {
				return reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateValidateTicketResultMessage{
					Result:  proto.Int32(1),
					Npid:    &clientId,
					GroupID: proto.Int32(1),
				})
			}

			// Set the server ID
			session.ServerId = serverId

			// Heartbeat detection
			/*time.AfterFunc(time.Minute*2, func() {
				if session != nil && session.Valid && session.Username != "" && !session.IsUnclean {
					if session.LastCI.IsZero() { //Before(time.Now().Truncate(time.Minute)) {
						err = utils.BanUser(session.Username, 10000, time.Hour*24*14)
						if err != nil {
							logger.Warning(err)
						}

						err = utils.KickUser(serverId, session.Npid, 10000)
						if err != nil {
							logger.Warning(err)
						}

						session.IsUnclean = true
					}
				}
			})*/

			// Reply that everything is fine
			return reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateValidateTicketResultMessage{
				Result:  proto.Int32(0),
				Npid:    &clientId,
				GroupID: proto.Int32(1),
			})
		}
	}

	// Wrong version or wrong NPID
	return reply.Reply(conn, packet_data.Header.Id, &protocol.AuthenticateValidateTicketResultMessage{
		Result:  proto.Int32(1),
		Npid:    &clientId,
		GroupID: proto.Int32(1),
	})
}
