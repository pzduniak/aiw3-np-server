package utils

import (
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"git.cloudrack.io/aiw3/np-server/environment"
	"git.cloudrack.io/aiw3/np-server/np/reply"
	"git.cloudrack.io/aiw3/np-server/np/storage"
	"git.cloudrack.io/aiw3/np-server/protocol/auth"
	"github.com/pzduniak/logger"
	"strconv"
	"time"
)

func BanUser(username string, reason int64, duration time.Duration) error {
	err := environment.Env.Database.Query(
		`INSERT INTO misago_ban(test, ban, reason_user, expires) VALUES(?, ?, ?, ?)`,
		0,        // ban by username
		username, // the username
		"Cheat detected ("+strconv.FormatInt(reason, 10)+")", // the reason
		time.Now().Add(duration),                             // ban for two weeks
	).Run()

	if err != nil {
		return err
	}

	return nil
}

func AddDelayedBan(npid uint64, reason int64, guid int64) error {
	logger.Debug("Fuck the police coming straight from the underground")
	logger.Debug("A young nigga got it bad cause I'm brown")
	return nil
}

var ServerDisappeared = errors.New("Server has disappeared")

func KickUser(serverID uint64, clientID uint64, reason int64) error {
	data := storage.GetServerConnection(serverID)

	if data != nil {
		return reply.Reply(data.Connection, 0, &auth.AuthenticateKickUserMessage{
			Npid:         proto.Uint64(clientID),
			Reason:       proto.Int32(1),
			ReasonString: proto.String("Cheat detected (" + strconv.FormatInt(reason, 10) + ")"),
		})
	}

	return ServerDisappeared
}
