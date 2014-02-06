package handlers

import (
	"bytes"
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/protocol/storage"
	"github.com/pzduniak/logger"
	"github.com/pzduniak/utility"
	"hash/crc32"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

const STAFF_RANK = 1

func RPCStorageGetUserFileMessage(conn net.Conn, connection_data *structs.ConnData, packet_data *structs.PacketData) error {
	// Unmarshal the request
	msg := new(storage.StorageGetUserFileMessage)
	err := proto.Unmarshal(packet_data.Content, msg)

	// Reply with an error message if something's wrong
	if err != nil {
		logger.Debugf("Error when unpacking a protobuf message; %s", err)
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageUserFileMessage{
			Result:   proto.Int32(3),
			FileName: msg.FileName,
			Npid:     msg.Npid,
			FileData: []byte(""),
		})
	}

	// If the user isn't authenticated or has wrong Npid, reply with an error
	if !connection_data.Authenticated ||
		connection_data.Npid != msg.GetNpid() {
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageUserFileMessage{
			Result:   proto.Int32(2),
			FileName: msg.FileName,
			Npid:     msg.Npid,
			FileData: []byte(""),
		})
	}

	// Generate
	filename := strconv.Itoa(structs.NpidToId(msg.GetNpid())) + "_" + strings.Trim(strings.Replace(msg.GetFileName(), "\\", "/", -1), "/")
	filepath := filepath.Join(environment.Env.Config.NP.UserFilesPath, filename)

	// File does not exist, abort the mission
	if !utility.FileExists(filepath) {
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageUserFileMessage{
			Result:   proto.Int32(2),
			FileName: msg.FileName,
			Npid:     msg.Npid,
			FileData: []byte(""),
		})
	}

	// Read from the file
	filecontents, err := ioutil.ReadFile(filepath)
	if err != nil {
		// Error? Log to the console and reply with an error message
		logger.Warningf("Error when reading a file in packet 1102; %s", err)
		return reply.Reply(conn, packet_data.Header.Id, &storage.StorageUserFileMessage{
			Result:   proto.Int32(3),
			FileName: msg.FileName,
			Npid:     msg.Npid,
			FileData: []byte(""),
		})
	}

	if msg.GetFileName() == "iw4.stat" {
		// 32bit uint32 le
		rd := bytes.NewReader(filecontents[2068:2072])

		var prestige uint32
		err = binary.Read(rd, binary.LittleEndian, &prestige)
		if err != nil {
			logger.Warning(err)
			return err
		}

		if connection_data.RankId == STAFF_RANK {
			filecontents[2068] = 11
		} else if prestige >= 11 {
			filecontents[2068] = 10
		}

		checksum := crc32.ChecksumIEEE(filecontents[4:])

		checksum_bytes := make([]byte, 4)
		checksum_buffer := bytes.NewBuffer(checksum_bytes)

		err = binary.Write(checksum_buffer, binary.LittleEndian, checksum)
		if err != nil {
			logger.Warning(err)
			return err
		}

		filecontents[0] = checksum_bytes[0]
		filecontents[1] = checksum_bytes[1]
		filecontents[2] = checksum_bytes[2]
		filecontents[3] = checksum_bytes[3]
	}

	logger.Debugf("Sending file %s to %s", filepath, connection_data.Username)

	return reply.Reply(conn, packet_data.Header.Id, &storage.StorageUserFileMessage{
		Result:   proto.Int32(0),
		FileName: msg.FileName,
		Npid:     msg.Npid,
		FileData: filecontents,
	})
}
