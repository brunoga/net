package server

import "net"

// connAddrWrapper is a net.Conn wrapper that overrides LocalAddr and RemoteAddr
// methods. Useful when used in conjunction with net.Pipe().
type connAddrWrapper struct {
	net.Conn

	localAddr  net.Addr
	remoteAddr net.Addr
}

// newConnAddrWrapper creates a new connAddrWrapper that will have the given
// local and remote addresses but otherwise use the given conn object for
// all other methods.
func newConnAddrWrapper(conn net.Conn, localAddr, remoteAddr net.Addr) net.Conn {
	return &connAddrWrapper{
		Conn:       conn,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

func (c *connAddrWrapper) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *connAddrWrapper) RemoteAddr() net.Addr {
	return c.remoteAddr
}
