//go:build darwin
// +build darwin

package poller

import (
	"syscall"
	"unsafe"
)

// KqueuePoller is a kqueue-based I/O multiplexer
type KqueuePoller struct {
	kqfd   int
	events []syscall.Kevent_t
}

// NewPoller creates a new Poller (macOS)
func NewPoller() (Poller, error) {
	kqfd, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}

	return &KqueuePoller{
		kqfd:   kqfd,
		events: make([]syscall.Kevent_t, 1024),
	}, nil
}

// Add adds a file descriptor to the watch list
func (p *KqueuePoller) Add(fd int) error {
	ev := syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		// Use level-triggered (default) for reliability
		// EV_CLEAR (edge-triggered) can miss events if not handled carefully
		Flags: syscall.EV_ADD | syscall.EV_ENABLE,
	}

	_, err := syscall.Kevent(p.kqfd, []syscall.Kevent_t{ev}, nil, nil)
	return err
}

// Remove removes a file descriptor from the watch list
func (p *KqueuePoller) Remove(fd int) error {
	ev := syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_DELETE,
	}

	_, err := syscall.Kevent(p.kqfd, []syscall.Kevent_t{ev}, nil, nil)
	return err
}

// Wait waits for I/O events
func (p *KqueuePoller) Wait(timeout int) ([]int, error) {
	var ts *syscall.Timespec
	if timeout >= 0 {
		ts = &syscall.Timespec{
			Sec:  int64(timeout / 1000),
			Nsec: int64((timeout % 1000) * 1000000),
		}
	}

	n, err := syscall.Kevent(p.kqfd, nil, p.events, ts)
	if err != nil && err != syscall.EINTR {
		return nil, err
	}

	// Handle negative or zero n
	if n <= 0 {
		return nil, nil
	}

	fds := make([]int, 0, n)
	for i := 0; i < n; i++ {
		fds = append(fds, int(p.events[i].Ident))
	}

	return fds, nil
}

// Close closes the Poller
func (p *KqueuePoller) Close() error {
	return syscall.Close(p.kqfd)
}

// SetNonblock sets non-blocking mode
func SetNonblock(fd int) error {
	return syscall.SetNonblock(fd, true)
}

// Unused variable to avoid compilation warnings
var _ = unsafe.Sizeof(0)
