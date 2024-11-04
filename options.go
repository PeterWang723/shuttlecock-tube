package shuttlecocktube

import (
	slog "log/slog"
	"time"
)

// Option is a functional option for the shuttlecock tube
type Option func(*Options)


// loadOptions applies the given functional options to a new Options instance and
// returns the result.
func loadOptions(opts ...Option) *Options {
	options := new(Options)
	for _, opt := range opts {
		opt(options)
	}
	return options
}	

// Options is the configuration options for the shuttlecock tube
type Options struct {
	// ExpiryDuration is the maximum lifetime of a task in the tube
	ExpiryDuration time.Duration

	// PreAlloc specifies whether the tube should pre-allocate goroutines
	PreAlloc bool

	//MaxBlockingTasks is the maximum number of tasks that can be submitted while the tube is blocked
	MaxBlockingTasks int

	// NonBlocking specifies whether the tube should block new tasks from being submitted
	NonBlocking bool

	// Logger is the logger to use for the tube
	Logger *slog.Logger
	// PanicHandler is the function to handle panics in the tube
	PanicHandler func(interface{})

	// DisablePurge disables the purge goroutine
	DisablePurge bool
}