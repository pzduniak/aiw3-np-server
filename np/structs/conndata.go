package structs

import (
	"net"
	"time"
)

type ConnData struct {
	Id            int
	Npid          uint64
	Username      string
	RankId        int
	Authenticated bool
	Token         string
	IsServer      bool
	IsUnclean     bool
	LastCI        time.Time
	ConnectionId  int
	ServerId      uint64
	ServerAddr    string
	PresenceData  map[string]string
	Connection    net.Conn
	Valid         bool
}
