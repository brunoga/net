package server

import (
	"fmt"
	"net"
	"testing"

	testing2 "github.com/brunoga/net/testing"
)

func TestNew(t *testing.T) {
	_, err := New("", "", nil)
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	c, err := New("", "", func(net.Conn) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client, got nil")
	}
}

func TestStart_TCP(t *testing.T) {
	s, err := New("tcp", "127.0.0.1:8080", func(net.Conn) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	s.listen = func(string, string) (net.Listener, error) {
		return nil, fmt.Errorf("listen error")
	}

	err = s.Start()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	ch := make(chan struct{})
	s.listen = func(string, string) (net.Listener, error) {
		return &testing2.MockListener{
			AcceptFunc: func() (net.Conn, error) {
				_ = <-ch

				return nil, fmt.Errorf("accept error")
			},
		}, nil
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	close(ch)

	s.wg.Wait()
}

func TestStart_UDP(t *testing.T) {
	s, err := New("udp", "127.0.0.1:8080", func(net.Conn) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	s.listenPacket = func(string, string) (net.PacketConn, error) {
		return nil, fmt.Errorf("listenPacket error")
	}

	err = s.Start()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	ch := make(chan struct{})
	s.listenPacket = func(string, string) (net.PacketConn, error) {
		return &testing2.MockPacketConn{
			ReadFromFunc: func(b []byte) (int, net.Addr, error) {
				<-ch
				return 0, nil, fmt.Errorf("readFrom error")
			},
			WriteToFunc: func(b []byte, addr net.Addr) (int, error) {
				<-ch
				return 0, fmt.Errorf("writeTo error")
			},
		}, nil
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = s.Start()
	if err == nil {
		t.Errorf("expected non-nil error, got nil")
	}

	// Unblock ReadFrom/WriteTo in the MockPacketConn.
	close(ch)

	s.wg.Wait()
}

func TestStop_TCP(t *testing.T) {
	s, err := New("tcp", "127.0.0.1:8080", func(net.Conn) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	ch := make(chan struct{})
	s.listen = func(string, string) (net.Listener, error) {
		return &testing2.MockListener{
			AcceptFunc: func() (net.Conn, error) {
				_ = <-ch

				return nil, fmt.Errorf("accept error")
			},
			CloseFunc: func() error {
				close(ch)

				return nil
			},
		}, nil
	}

	err = s.Stop()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = s.Stop()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestStop_UDP(t *testing.T) {
	s, err := New("udp", "127.0.0.1:8080", func(net.Conn) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	ch := make(chan struct{})
	s.listenPacket = func(string, string) (net.PacketConn, error) {
		return &testing2.MockPacketConn{
			ReadFromFunc: func(b []byte) (int, net.Addr, error) {
				<-ch
				return 0, nil, fmt.Errorf("readFrom error")
			},
			WriteToFunc: func(b []byte, addr net.Addr) (int, error) {
				<-ch
				return 0, fmt.Errorf("writeTo error")
			},
			CloseFunc: func() error {
				close(ch)
				return nil
			},
		}, nil
	}

	err = s.Stop()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = s.Stop()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestConnection_TCP(t *testing.T) {
	handlerCh := make(chan []byte)
	connectionHandler := func(conn net.Conn) {
		buffer := make([]byte, 4096)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				break
			}

			handlerCh <- buffer[:n]
		}

		conn.Close()
	}

	s, err := New("tcp", "", connectionHandler)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	connCh := make(chan net.Conn)
	s.listen = func(string, string) (net.Listener, error) {
		return &testing2.MockListener{
			AcceptFunc: func() (net.Conn, error) {
				conn := <-connCh
				if conn == nil {
					return nil, fmt.Errorf("accept error")
				}

				return conn, nil
			},
			CloseFunc: func() error {
				close(connCh)
				return nil
			},
		}, nil
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	localConn, remoteConn := net.Pipe()

	connCh <- remoteConn

	localConn.Write([]byte("hello"))

	data := <-handlerCh
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %v", string(data))
	}

	defer s.Stop()
}

func TestConnection_UDP(t *testing.T) {
	handlerCh := make(chan []byte)
	connectionHandler := func(conn net.Conn) {
		if conn.LocalAddr().String() != "a.a.a.a:bbbb" {
			t.Errorf("expected 'a.a.a.a:bbbb', got %v", conn.LocalAddr().String())
		}

		if conn.RemoteAddr().String() != "x.x.x.x:yyyy" {
			t.Errorf("expected 'x.x.x.x:yyy', got %v", conn.RemoteAddr().String())
		}

		buffer := make([]byte, 4096)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				break
			}

			handlerCh <- buffer[:n]

			_, err = conn.Write([]byte("hello yourself"))
			if err != nil {
				break
			}
		}

		conn.Close()
	}

	s, err := New("udp", "", connectionHandler)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	type ConnData struct {
		Addr net.Addr
		Data []byte
	}
	readFromDataCh := make(chan *ConnData)
	writeToDataCh := make(chan *ConnData)

	packetConn := &testing2.MockPacketConn{
		ReadFromFunc: func(b []byte) (int, net.Addr, error) {
			connData := <-readFromDataCh
			if connData == nil {
				return 0, nil, fmt.Errorf("readfrom error")
			}

			copy(b, connData.Data)

			return len(connData.Data), connData.Addr, nil
		},
		WriteToFunc: func(b []byte, addr net.Addr) (int, error) {
			defer func() {
				writeToDataCh <- &ConnData{
					Addr: addr,
					Data: b,
				}
			}()

			return len(b), nil
		},
		CloseFunc: func() error {
			close(readFromDataCh)
			return nil
		},
		LocalAddrFunc: func() net.Addr {
			return &testing2.MockAddr{
				NetworkFunc: func() string {
					return "udp"
				},
				StringFunc: func() string {
					return "a.a.a.a:bbbb"
				},
			}
		},
	}

	s.listenPacket = func(string, string) (net.PacketConn, error) {
		return packetConn, nil
	}

	err = s.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	readFromDataCh <- &ConnData{
		Addr: &testing2.MockAddr{
			NetworkFunc: func() string {
				return "udp"
			},
			StringFunc: func() string {
				return "x.x.x.x:yyyy"
			},
		},
		Data: []byte("hello"),
	}

	data := <-handlerCh

	if string(string(data)) != "hello" {
		t.Errorf("expected 'hello', got %v", string(data))
	}

	writeToData := <-writeToDataCh
	if writeToData.Addr.Network() != "udp" {
		t.Errorf("expected 'udp', got %v", writeToData.Addr.Network())
	}
	if writeToData.Addr.String() != "x.x.x.x:yyyy" {
		t.Errorf("expected 'x.x.x.x:yyyy', got %v", writeToData.Addr.String())
	}
	if string(writeToData.Data) != "hello yourself" {
		t.Errorf("expected 'hello yourself', got %v", string(writeToData.Data))
	}

	defer s.Stop()
}
