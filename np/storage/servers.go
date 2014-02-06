package storage

import (
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"sync"
)

var Servers map[uint64]*structs.ConnData
var servers_mutex = new(sync.Mutex)

func init() {
	Servers = make(map[uint64]*structs.ConnData)
}

func SetServerConnection(npid uint64, data *structs.ConnData) {
	servers_mutex.Lock()
	Servers[npid] = data
	servers_mutex.Unlock()
}

func GetServerConnection(npid uint64) *structs.ConnData {
	if a, ok := Servers[npid]; ok {
		return a
	}

	return nil
}

func DeleteServerConnection(npid uint64) {
	servers_mutex.Lock()
	delete(Servers, npid)
	servers_mutex.Unlock()
}
