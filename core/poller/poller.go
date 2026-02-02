package poller

// Poller is the I/O multiplexing interface
type Poller interface {
	Add(fd int) error
	Remove(fd int) error
	Wait(timeout int) ([]int, error)
	Close() error
}
