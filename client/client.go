package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

// Client is a client that connects to a specific address using a specific
// network and manages sending and receioving that through that connection. It
// supports any protocols that implement the net.Conn interface.
type Client struct {
	network     string
	address     string
	splitFunc   bufio.SplitFunc
	receiveFunc ReceiveFunc

	// For testing purposes only.
	dial func(string, string) (net.Conn, error)

	wg sync.WaitGroup

	m       sync.Mutex
	conn    net.Conn
	started bool
}

// ReceiveFunc is the signature for functions that should be called when
// data is received from the server and a valid token is detected by the
// associated bufio.SplitFunc.
type ReceiveFunc func(data []byte)

// ScanFullBuffer is a bufio.SplitFunc that always returns all data in the
// buffer.
func ScanFullBuffer(data []byte, atEOF bool) (int, []byte, error) {
	var err error
	if atEOF {
		err = bufio.ErrFinalToken
	}

	return len(data), data, err
}

// New creates a new Client instance that will try to connect to the given
// network and address and that will use the given splitFunc to parse incoming
// data into tokens and the call the given receiveFunc to handle those tokens.
// Note that New does not do any validation whatsoever and errors will be
// reported when Start is called.
func New(network, address string, splitFunc bufio.SplitFunc,
	receiveFunc ReceiveFunc) (*Client, error) {
	if receiveFunc == nil {
		return nil, fmt.Errorf("receiveFunc cannot be nil")
	}

	if splitFunc == nil {
		return nil, fmt.Errorf("splitFunc cannot be nil")
	}

	return &Client{
		network:     network,
		address:     address,
		splitFunc:   splitFunc,
		receiveFunc: receiveFunc,
		dial:        net.Dial,
	}, nil
}

// Start tries to start the connection associated with this Client. It returns a
// nil error on success and a non-nil error on failure.
func (c *Client) Start() error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.started {
		return fmt.Errorf("client already started")
	}

	conn, err := c.dial(c.network, c.address)
	if err != nil {
		return err
	}

	c.conn = conn

	c.wg.Add(1)
	go c.receiveLoop()

	c.started = true

	return nil
}

// Stop tries to stop (close) the connection associated with this Client. It
// returns a nil error on success and a non-nil error on failure.
func (c *Client) Stop() error {
	c.m.Lock()
	defer c.m.Unlock()

	if !c.started {
		return fmt.Errorf("client not started")
	}

	c.conn.Close()

	c.wg.Wait()

	c.started = false
	c.conn = nil

	return nil
}

// Send tries to send the given data to the connection associated with this
// Client. It returns a nil error on success and a non-nil error on failure.
func (c *Client) Send(data []byte) error {
	c.m.Lock()
	defer c.m.Unlock()

	if !c.started {
		return fmt.Errorf("client not started")
	}

	_, err := c.conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) receiveLoop() {
	scanner := bufio.NewScanner(c.conn)
	scanner.Split(c.splitFunc)
	for scanner.Scan() {
		c.receiveFunc(scanner.Bytes())
	}

	c.wg.Done()
}
