package testing

import (
	"fmt"
	"net"
	"time"
)

// MockPacketConn is a mockable implementation of net.PacketConn.
type MockPacketConn struct {
	ReadFromFunc         func(b []byte) (int, net.Addr, error)
	WriteToFunc          func(b []byte, addr net.Addr) (int, error)
	CloseFunc            func() error
	LocalAddrFunc        func() net.Addr
	SetDeadlineFunc      func(t time.Time) error
	SetReadDeadlineFunc  func(t time.Time) error
	SetWriteDeadlineFunc func(t time.Time) error
}

func (m *MockPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if m.ReadFromFunc != nil {
		return m.ReadFromFunc(b)
	}

	return 0, nil, fmt.Errorf("readfrom error")
}

func (m *MockPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if m.WriteToFunc != nil {
		return m.WriteToFunc(b, addr)
	}

	return 0, fmt.Errorf("writeto error")
}

func (m *MockPacketConn) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return fmt.Errorf("close error")
}

func (m *MockPacketConn) LocalAddr() net.Addr {
	if m.LocalAddrFunc != nil {
		return m.LocalAddrFunc()
	}

	return nil
}

func (m *MockPacketConn) SetDeadline(t time.Time) error {
	if m.SetDeadlineFunc != nil {
		return m.SetDeadlineFunc(t)
	}

	return fmt.Errorf("setdeadline error")
}

func (m *MockPacketConn) SetReadDeadline(t time.Time) error {
	if m.SetReadDeadlineFunc != nil {
		return m.SetReadDeadlineFunc(t)
	}

	return fmt.Errorf("setreaddeadline error")
}

func (m *MockPacketConn) SetWriteDeadline(t time.Time) error {
	if m.SetWriteDeadlineFunc != nil {
		return m.SetWriteDeadlineFunc(t)
	}

	return fmt.Errorf("setwritedeadline error")
}
