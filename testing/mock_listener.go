package testing

import (
	"fmt"
	"net"
)

// MockListener is a mockable implementation of net.Listener.
type MockListener struct {
	AcceptFunc func() (net.Conn, error)
	CloseFunc  func() error
	AddrFunc   func() net.Addr
}

func (m *MockListener) Accept() (net.Conn, error) {
	if m.AcceptFunc != nil {
		return m.AcceptFunc()
	}

	return nil, fmt.Errorf("accept error")
}

func (m *MockListener) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return fmt.Errorf("close error")
}

func (m *MockListener) Addr() net.Addr {
	if m.AddrFunc != nil {
		return m.AddrFunc()
	}

	return nil
}
