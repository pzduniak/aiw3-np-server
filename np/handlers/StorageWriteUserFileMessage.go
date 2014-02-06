package handlers

import (
	"code.google.com/p/goprotobuf/proto"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/protocol/storage"
	"github.com/pzduniak/logger"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

func RPCStorageWriteUserFileMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the data
	msg := new(storage.StorageWriteUserFileMessage)
	err := proto.Unmarshal(packet_data.Content, msg)
	if err != nil {
		return err
	}

	// If the user is not authenticated or the Npid is spoofed,
	// return an error (code 1)
	if connection_data.Authenticated == false ||
		connection_data.Npid != msg.GetNpid() {
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageWriteUserFileResultMessage{
			Result:   proto.Int32(1),
			FileName: msg.FileName,
			Npid:     msg.Npid,
		})
	}

	// Generate file paths
	filename := strconv.Itoa(structs.NpidToId(msg.GetNpid())) + "_" + strings.Trim(strings.Replace(msg.GetFileName(), "\\", "/", -1), "/")
	filepath := filepath.Join(environment.Env.Config.NP.UserFilesPath, filename)

	// Write a file
	err = ioutil.WriteFile(filepath, msg.GetFileData(), 0644)
	if err != nil {
		// IO error - log it to the console and return an error
		logger.Warningf("Can't save user file; %s", err)
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageWriteUserFileResultMessage{
			Result:   proto.Int32(2),
			FileName: msg.FileName,
			Npid:     msg.Npid,
		})
	}

	// Reply that we have successfully saved the file
	return reply.Reply(conn, packet_data.Header.Id, &storage.StorageWriteUserFileResultMessage{
		Result:   proto.Int32(0),
		FileName: msg.FileName,
		Npid:     msg.Npid,
	})
}
