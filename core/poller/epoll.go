//go:build linux
// +build linux

package poller

import (
	"syscall"
	"unsafe"
)

// EpollPoller is an epoll-based I/O multiplexer
type EpollPoller struct {
	epfd   int
	events []syscall.EpollEvent
}

// NewPoller creates a new Poller (Linux)
func NewPoller() (Poller, error) {
	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &EpollPoller{
		epfd:   epfd,
		events: make([]syscall.EpollEvent, 1024),
	}, nil
}

// Add adds a file descriptor to the watch list
func (p *EpollPoller) Add(fd int) error {
	ev := syscall.EpollEvent{
		// EPOLLIN: Read events
		// EPOLLRDHUP (0x2000): Detect peer shutdown
		// Use level-triggered (default, no EPOLLET) for reliability
		Events: uint32(syscall.EPOLLIN) | uint32(0x2000),
		Fd:     int32(fd),
	}

	return syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_ADD, fd, &ev)
}

// Remove removes a file descriptor from the watch list
func (p *EpollPoller) Remove(fd int) error {
	return syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
}

// Wait waits for I/O events
func (p *EpollPoller) Wait(timeout int) ([]int, error) {
	n, err := syscall.EpollWait(p.epfd, p.events, timeout)
	if err != nil && err != syscall.EINTR {
		return nil, err
	}

	// Handle negative or zero n
	if n <= 0 {
		return nil, nil
	}

	fds := make([]int, 0, n)
	for i := 0; i < n; i++ {
		fds = append(fds, int(p.events[i].Fd))
	}

	return fds, nil
}

// Close closes the Poller
func (p *EpollPoller) Close() error {
	return syscall.Close(p.epfd)
}

// SetNonblock sets non-blocking mode
func SetNonblock(fd int) error {
	return syscall.SetNonblock(fd, true)
}

// Unused variable to avoid compilation warnings
var _ = unsafe.Sizeof(0)
