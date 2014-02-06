package storage

import (
	"git.cloudrack.io/aiw3/np-server/np/structs"
	"sync"
)

var clients map[uint64]*structs.ConnData
var clients_mutex = new(sync.Mutex)

func init() {
	clients = make(map[uint64]*structs.ConnData)
}

func SetClientConnection(npid uint64, data *structs.ConnData) {
	clients_mutex.Lock()
	clients[npid] = data
	clients_mutex.Unlock()
}

func GetClientConnection(npid uint64) *structs.ConnData {
	if a, ok := clients[npid]; ok {
		return a
	}

	return nil
}

func DeleteClientConnection(npid uint64) {
	clients_mutex.Lock()
	delete(clients, npid)
	clients_mutex.Unlock()
}
