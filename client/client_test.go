package client

import (
	"net"
	"testing"
)

func TestNew(t *testing.T) {
	_, err := New("", "", nil, func([]byte) {})
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	_, err = New("", "", ScanFullBuffer, nil)
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	c, err := New("", "", ScanFullBuffer, func([]byte) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client, got nil")
	}
}

func TestStart(t *testing.T) {
	c, err := New("", "", ScanFullBuffer, func([]byte) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	c.dial = func(network, address string) (net.Conn, error) {
		return nil, &net.AddrError{}
	}

	err = c.Start()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	c.dial = func(network, address string) (net.Conn, error) {
		return clientConn, nil
	}

	err = c.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = c.Start()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	serverConn.Close()

	c.wg.Wait()
}

func TestStop(t *testing.T) {
	c, err := New("", "", ScanFullBuffer, func([]byte) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer func() {
		clientConn.Close()
		serverConn.Close()
	}()

	err = c.Stop()
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	c.dial = func(network, address string) (net.Conn, error) {
		return clientConn, nil
	}

	err = c.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = c.Stop()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestSend(t *testing.T) {
	c, err := New("", "", ScanFullBuffer, func([]byte) {})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = c.Send([]byte("test"))
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	c.dial = func(network, address string) (net.Conn, error) {
		return clientConn, nil
	}

	err = c.Start()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	defer c.Stop()

	ch := make(chan []byte)
	go func() {
		buff := make([]byte, 1024)
		n, err := serverConn.Read(buff)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}

		ch <- buff[:n]
	}()

	err = c.Send([]byte("test"))
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	data := <-ch
	if string(data) != "test" {
		t.Errorf("expected %v, got %v", "test", string(data))
	}

	c.conn.Close()

	err = c.Send([]byte("test"))
	if err == nil {
		t.Error("expected non-nil error, got nil")
	}
}
