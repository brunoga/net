package server

import (
	"fmt"
	"net"
	"sync"
)

// Server is a server that handles both packet and stream protocols with the
// same API. There is some complexity in achieving that so I would not use
// it for performance-critical servers (that is being said without any
// performance testing being done whatsoever). Clients of the API just need
// to provide a connection handler and just read/write data from/to the
// associated connection (in exactly the same way for underlying stream or
// packet connections).
type Server struct {
	network           string
	address           string
	connectionHandler ConnectionHandler

	// For testing purposes only.
	listen       func(string, string) (net.Listener, error)
	listenPacket func(string, string) (net.PacketConn, error)

	packetAddrConnMap map[string]net.Conn

	wg sync.WaitGroup

	m          sync.Mutex
	listener   net.Listener   // nil if packetConn is not
	packetConn net.PacketConn // nil if listener is not
	started    bool
}

// ConnectionHandler is the signature for functions that that will handle
// server connections. The handler is called on its own goroutine.
type ConnectionHandler func(net.Conn)

// New creates a new Server instance that will try to listen at the given
// network and address and that will call the given connectionHandler to handle
// incoming connections (and "fake" connections for packet connections).
// Note that New only validates that connectionHandler is not nil. All other
// errors will be reported when Start is called.
func New(network, address string,
	connectionHandler ConnectionHandler) (*Server, error) {
	if connectionHandler == nil {
		return nil, fmt.Errorf("connectionHandler cannot be nil")
	}

	return &Server{
		network:           network,
		address:           address,
		connectionHandler: connectionHandler,
		listen:            net.Listen,
		listenPacket:      net.ListenPacket,
		packetAddrConnMap: make(map[string]net.Conn),
	}, nil
}

// Start tries to start listening for incoming connections. It returns a nil
// error on success and a non-nil error on failure.
func (s *Server) Start() error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	switch s.network {
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		listener, err := s.listen(s.network, s.address)
		if err != nil {
			return err
		}

		s.listener = listener

		s.wg.Add(1)
		go s.listenLoop()
	default:
		packetConn, err := s.listenPacket(s.network, s.address)
		if err != nil {
			return err
		}

		s.packetConn = packetConn

		s.wg.Add(1)
		go s.packetListenLoop()
	}

	s.started = true

	return nil
}

// Stop tries to stop listening for connections. It returns a nil error on
// success and a non-nil error on failure.
func (s *Server) Stop() error {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.started {
		return fmt.Errorf("server not started")
	}

	// Signal listener goroutines to exit.
	if s.listener != nil {
		s.listener.Close()
	} else if s.packetConn != nil {
		s.packetConn.Close()
	}

	// Unlock while we wait for the gotoutines to cleanup.
	s.m.Unlock()

	s.wg.Wait()

	// Lock again to set started to false and aslo satisfy the defer above.
	// TODO(bga): Revisit this.
	s.m.Lock()

	s.started = false

	return nil
}

func (s *Server) listenLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				continue
			}
			break
		}

		go s.connectionHandler(conn)
	}

	s.listener = nil

	s.wg.Done()
}

func (s *Server) packetListenLoop() {
	for {
		buffer := make([]byte, 4096)
		n, addr, err := s.packetConn.ReadFrom(buffer)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				continue
			}
			break
		}

		var localConn, remoteConn net.Conn
		var ok bool
		if localConn, ok = s.packetAddrConnMap[addr.String()]; !ok {
			localConn, remoteConn = net.Pipe()
			s.packetAddrConnMap[addr.String()] = localConn

			// Read from localConn and relay data to packetConn.
			s.wg.Add(1)
			go s.relayLoop(localConn, addr)

			go s.connectionHandler(newConnAddrWrapper(remoteConn,
				s.packetConn.LocalAddr(), addr))
		}

		_, err = localConn.Write(buffer[:n])
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				continue
			}

			break
		}
	}

	for _, conn := range s.packetAddrConnMap {
		conn.Close()
	}

	s.packetAddrConnMap = nil
	s.packetConn = nil

	s.wg.Done()
}

func (s *Server) relayLoop(localConn net.Conn, addr net.Addr) {
	buffer := make([]byte, 4096)
	for {
		n, err := localConn.Read(buffer)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				continue
			}
			break
		}

		_, err = s.packetConn.WriteTo(buffer[:n], addr)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Temporary() {
				continue
			}
			break
		}
	}

	localConn.Close()
	delete(s.packetAddrConnMap, addr.String())

	s.wg.Done()
}
