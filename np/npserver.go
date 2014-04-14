package np

import (
	"bytes"
	"encoding/binary"
	"github.com/pzduniak/aiw3-np-server/environment"
	"github.com/pzduniak/aiw3-np-server/np/storage"
	"github.com/pzduniak/aiw3-np-server/np/structs"
	"github.com/pzduniak/logger"
	"io"
	"net"
	"time"
)

type NPServer struct {
	listener net.Listener
}

func New() *NPServer {
	return &NPServer{}
}

var TotalConnections = 0
var LastCid = 0

func (a *NPServer) Start() error {
	// You can't use property, err := sth, because golang will qq
	// This thing binds a classic TCP listener to the server
	var err error
	a.listener, err = net.Listen("tcp", environment.Env.Config.NP.BindingAddress)
	if err != nil {
		logger.Fatalf("Error listening; %s", err)
	}

	defer a.listener.Close()

	logger.Infof("Serving NP server on %s", environment.Env.Config.NP.BindingAddress)

	var tempDelay time.Duration

	for {
		conn, e := a.listener.Accept()

		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				logger.Errorf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			return e
		}

		// Seperated it from the listener code to make it clean
		go a.HandleConnection(conn)
	}

	return nil
}

const packet_signature = 0xDEADC0DE
const data_buffer_size = 256

func (n *NPServer) HandleConnection(conn net.Conn) {
	// Close the connection after it ends its execution
	defer conn.Close()

	// This is the place where you should put stuff to be executed for the "connection" event
	// The below code is the most non-thread-safe way to create such identifiers
	// If we get enough load, servers can fuck up
	TotalConnections++
	LastCid++
	cid := LastCid

	// Here comes a struct for connection data, that we pass over to handlers
	connection_data := new(structs.ConnData)
	connection_data.Authenticated = false                  // Because we don't want session stealing
	connection_data.ConnectionId = cid                     // Connection ID is mainly used in servers
	connection_data.PresenceData = make(map[string]string) // If we don't init it here, it'll panic later
	connection_data.Connection = conn
	connection_data.Valid = true

	for {
		// Set timeout
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// First 16 bytes are defined in the structs subpackage
		// It consists of four, 4-byte Little Endian unsigned 32bit integers
		// in the following order: signature, length, type, id
		headerBytes := make([]byte, 16)
		_, err := conn.Read(headerBytes)
		if err != nil {
			if err != io.EOF {
				logger.Warningf("Error while reading the header; %s", err)
			}

			break
		}

		// Initialize a bytes mapper
		buf := bytes.NewReader(headerBytes)

		// Map data to the struct
		var packet_header structs.PacketHeader
		err = binary.Read(buf, binary.LittleEndian, &packet_header)
		if err != nil {
			logger.Warningf("Error while mapping packet_header to struct; %s", err)
			continue
		}

		// Signature check
		if packet_header.Signature != packet_signature {
			logger.Warningf(
				"Signature doesn't match (from %s), received 0x%X",
				conn.RemoteAddr().String(),
				packet_header.Signature,
			)

			continue
		}

		// Length from the packet_header specifies how many of the next bytes are the content
		// Pass it over to the handlers which will decode it using protobuf
		contentBytes := make([]byte, packet_header.Length)

		// Set timeout
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Read the body
		_, err = io.ReadFull(conn, contentBytes)
		if err != nil {
			logger.Warningf("Error while reading; %s", err)
			break
		}

		// Log that we are parsing a RPC message
		logger.Debugf("Received message %d (ID: %d) from %s", packet_header.Type, packet_header.Id, structs.StripPort(conn.RemoteAddr().String()))

		// Generate a new PacketData struct
		packet_data := new(structs.PacketData)
		packet_data.Header = packet_header
		packet_data.Content = contentBytes

		// Pass it to the handlers. Not the best idea to do it using params, we should use
		// a struct, so we will easily be able to inject more variables.
		err = HandleMessage(conn, connection_data, packet_data)
		if err != nil {
			logger.Warningf("Error while handling message %d from %s; %s", packet_data.Header.Id, conn.RemoteAddr().String(), err)
		}
	}

	if connection_data.Authenticated {
		if connection_data.IsServer {
			storage.DeleteServerConnection(connection_data.Npid)
		} else {
			storage.DeleteClientConnection(connection_data.Npid)
		}

	}

	// I guess that count is going to be used by the web API
	TotalConnections--
	connection_data.Valid = false
}
