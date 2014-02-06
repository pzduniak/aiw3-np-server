package aci

import (
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"git.cloudrack.io/aiw3/np-server/utils"
	"github.com/pzduniak/logger"
	"net"
	"strconv"
	"time"
)

func HandleCI2(
	conn net.Conn,
	connection_data *structs.ConnData,
	packet_data *structs.PacketData,
	stringParts []string,
) error {
	if len(stringParts) != 2 {
		return nil
	}

	// I assume that's the classic anticheat
	// I have no idea if it is supposed to work, but I did it
	reason, err := strconv.Atoi(stringParts[1])
	if err != nil {
		return err
	}

	if reason == 50001 {
		connection_data.LastCI = time.Now()
	}

	if connection_data.IsUnclean {
		return nil
	}

	connection_data.IsUnclean = true

	// Write to database
	err = environment.Env.Database.Query(`
INSERT INTO 
	misago_ban(
		test,
		ban,
		reason_user,
		expires
	)
VALUES(
	?,
	?,
	?,
	?
)`, 0, connection_data.Username, "Cheat detected ("+stringParts[1]+")", time.Now().Add(time.Hour*24*14)).Run()
	if err != nil {
		return err
	}

	logger.Info(connection_data.Npid, " marked unclean for ", reason)

	if connection_data.ServerId != 0 {
		return utils.KickUser(connection_data.ServerId, connection_data.Npid, int64(reason))
	}

	return nil
}
