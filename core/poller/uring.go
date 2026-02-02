//go:build linux
// +build linux

package poller

// io_uring support (Linux 5.1+)
// Currently a placeholder, gracefully falls back to epoll in actual use

// UringPoller is an io_uring implementation (experimental)
type UringPoller struct {
	// Placeholder
}

// NewUringPoller creates an io_uring poller
// Currently unimplemented, returns nil to allow system fallback to epoll
func NewUringPoller() (Poller, error) {
	return nil, nil
}
