package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/protocol"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"github.com/pzduniak/logger"
	"github.com/pzduniak/utility"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
)

// RPC command #1102
// Executed at application's startup to verify that the server is alive
// Also might be used for anticheat. Not sure.
// Acts as a simple file server over the Protobuf-based protocol
func RPCStorageGetPublisherFileMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the message
	msg := new(protocol.StorageGetPublisherFileMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// Generate the filepath
	filename := filepath.Join(environment.Env.Config.NP.PubFilesPath, strings.Trim(strings.Replace(msg.GetFileName(), "\\", "/", -1), "/"))

	// Make sure the file exists. If it doesn't, return a reply with an error
	if !utility.FileExists(filename) {
		return reply.Reply(conn, packet_data.Header.Id, &protocol.StoragePublisherFileMessage{
			Result:   proto.Int32(1),
			FileName: msg.FileName,
			FileData: []byte(""),
		})
	}

	// Read the data
	filecontents, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Debugf("Error when reading a file in packet 1101; %s", err)
		return reply.Reply(conn, packet_data.Header.Id, &protocol.StoragePublisherFileMessage{
			Result:   proto.Int32(3),
			FileName: msg.FileName,
			FileData: []byte(""),
		})
	}

	// Debug some info
	logger.Debugf("Sending file %s to %s", filename, connection_data.Username)

	// Reply with data
	return reply.Reply(conn, packet_data.Header.Id, &protocol.StoragePublisherFileMessage{
		Result:   proto.Int32(0),
		FileName: msg.FileName,
		FileData: filecontents,
	})
}
