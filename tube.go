package shuttlecocktube

import (
	"context"
	"sync"
	"sync/atomic"
)

type tubeCommon struct {
	// capacity is the capacity of the tube
	capacity int32

	// running is the number of currently running goroutines
	running int32

	// lock is the lock for the tube
	lock sync.Locker

	// workers is the worker queue
	workers workerQueue

	// state is the state of the tube
	state int32

	// cond is waiting for a idle worker
	cond *sync.Cond

	// allDone is the channel for waiting for all goroutines to finish
	allDone chan struct{}

	// once makes sure the tube is only released once
	once *sync.Once

	// workerCache is the cache for workers
	workerCache sync.Pool

	// waiting is the number of waiting goroutines
	waiting int32

	// purgeDone makes sure the tube is only purged 
	purgeDone int32

	// purgeCtx is the context for purging the tube
	purgeCtx  context.Context

	// stopPurge is the cancel function for purging the tube
	stopPurge context.CancelFunc

	// ticktockDone makes sure the tube is only ticked tocked
	ticktockDone int32

	// ticktockCtx is the context for ticktocking the tube
	ticktockCtx  context.Context

	// stopTicktock is the cancel function for ticktocking the tube
	stopTicktock context.CancelFunc

	// now is the current time
	now atomic.Value

	// options is the options for the tube
	options *Options
}

// Tube accepts tasks and process them simultaneously
// limit goroutines to a certain number by recycling them
type Tube struct {
	tubeCommon
}

// NewTube creates a new tube with the given size and options
// The tube can be configured with several options:
//   - PreAlloc: whether to preallocate the goroutine pool
//   - ExpiryDuration: the duration after which a goroutine is purged
//   - DisablePurge: whether to disable the purge goroutine
//   - Logger: the logger to use for logging
//   - MaxBlockingTasks: the maximum number of blocking tasks
//   - NonBlocking: whether to use blocking or non-blocking goroutines
func NewTube(size int, options ...Option) (*Tube, error) {
	if size <= 0 {
		size = -1
	}

	opts := loadOptions(options...)

	// check if the tube options are valid
	if !opts.DisablePurge{
		if expiry := opts.ExpiryDuration; expiry < 0 {
			return nil, ErrIvalidTubeExpiry
		} else if expiry == 0 {
			opts.ExpiryDuration = DefaultCleanInterval
		}
	}

	// use the default logger if none is provided
	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	// create the tube
	t := &Tube{
		tubeCommon: tubeCommon{
			capacity: int32(size),
			lock:     &sync.Mutex{},
			allDone:  make(chan struct{}),
			once:     &sync.Once{},
			options: opts,
		}}

	// create the worker cache
	t.workerCache.New = func() interface{} {
		return &goWorker {
			tube: t,
			task: make(chan func(), workerChanCap),
		}
	}

	// create the worker queue
	if t.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		t.workers = newWorkerQueue(queueTypeLoopQueue, size)
	} else {
		t.workers = newWorkerQueue(queueTypeStack, 0)
	}

	// create the condition variable
	t.cond = sync.NewCond(t.lock)

	// start the purge goroutine
	t.goPurge()
	
	// start the ticktock goroutine
	t.goTicktock()

	return t, nil
}
