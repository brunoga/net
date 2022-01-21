package testing

// MockAddr is a mockable implementation of net.Addr.
type MockAddr struct {
	NetworkFunc func() string
	StringFunc  func() string
}

func (m *MockAddr) Network() string {
	if m.NetworkFunc != nil {
		return m.NetworkFunc()
	}

	return "tcp"
}

func (m *MockAddr) String() string {
	if m.StringFunc != nil {
		return m.StringFunc()
	}

	return "127.0.0.1:8080"
}
