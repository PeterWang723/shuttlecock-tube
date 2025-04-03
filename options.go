package shuttlecocktube

import (
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
	Logger Logger
	// PanicHandler is the function to handle panics in the tube
	PanicHandler func(interface{})

	// DisablePurge disables the purge goroutine
	DisablePurge bool
}

// WithOptions accepts the whole options config.
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithExpiryDuration sets up the interval time of cleaning up goroutines.
func WithExpiryDuration(expiryDuration time.Duration) Option {
	return func(opts *Options) {
		opts.ExpiryDuration = expiryDuration
	}
}

// WithPreAlloc indicates whether it should malloc for workers.
func WithPreAlloc(preAlloc bool) Option {
	return func(opts *Options) {
		opts.PreAlloc = preAlloc
	}
}

// WithMaxBlockingTasks sets up the maximum number of goroutines that are blocked when it reaches the capacity of pool.
func WithMaxBlockingTasks(maxBlockingTasks int) Option {
	return func(opts *Options) {
		opts.MaxBlockingTasks = maxBlockingTasks
	}
}

// WithNonblocking indicates that pool will return nil when there is no available workers.
func WithNonblocking(nonblocking bool) Option {
	return func(opts *Options) {
		opts.NonBlocking = nonblocking
	}
}

// WithPanicHandler sets up panic handler.
func WithPanicHandler(panicHandler func(any)) Option {
	return func(opts *Options) {
		opts.PanicHandler = panicHandler
	}
}

// WithLogger sets up a customized logger.
func WithLogger(logger Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithDisablePurge indicates whether we turn off automatically purge.
func WithDisablePurge(disable bool) Option {
	return func(opts *Options) {
		opts.DisablePurge = disable
	}
}