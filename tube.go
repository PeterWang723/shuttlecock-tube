package shuttlecocktube

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
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

type Logger interface {
	Printf(format string, args ...any)
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

// Cap returns the capacity of the tube.
func (t *Tube) Cap() int {
	return int(atomic.LoadInt32(&t.capacity))
}

func (t *Tube) addWaiting(delta int) {
	atomic.AddInt32(&t.waiting, int32(delta))
}
func (t *Tube) Release() {
		if !atomic.CompareAndSwapInt32(&t.state, OPENED, CLOSED) {
			return
		}

		if t.stopPurge != nil {
			t.stopPurge()
			t.stopPurge = nil
		}
		
		if t.stopTicktock != nil {
			t.stopTicktock()
			t.stopTicktock = nil
		}

		t.lock.Lock()
		t.workers.reset()
		t.lock.Unlock()
		// There might be some callers waiting in retrieveWorker(), so we need to wake them up to prevent
		// those callers blocking infinitely.
		t.cond.Broadcast()
}

func (t *Tube) nowTime() time.Time {
	return t.now.Load().(time.Time)
}

func (t *Tube) Submit(task func()) error {
	if t.IsClosed() {
		return ErrTubeClosed
	}

	w, err := t.retrieveWorker()
	if w != nil {
		w.inputFunc(task)
	}

	return err
}

func (t *Tube) retrieveWorker() (w worker, err error) {
	t.lock.Lock()

retry:
	// First try to fetch the worker from the queue.
	if w = t.workers.detach(); w != nil {
		t.lock.Unlock()
		return
	}

	if capacity := t.Cap(); capacity == -1 || capacity > t.Running() {
		t.lock.Unlock()
		w = t.workerCache.Get().(worker)
		w.run()
		return
	}

	if t.options.NonBlocking || (t.options.MaxBlockingTasks != 0 && t.Waiting() >= t.options.MaxBlockingTasks) {
		t.lock.Unlock()
		return nil, ErrTubeOverload
	}

	t.addWaiting(1)
	t.cond.Wait()
	t.addWaiting(-1)

	if t.IsClosed() {
		t.lock.Unlock()
		return nil, ErrTubeClosed
	}

	goto retry
}

// Running returns the number of currently running goroutines in the tube.
func (t *Tube) Running() int {
	return int(atomic.LoadInt32(&t.running))
}


func (t *Tube) IsClosed() bool {
	return atomic.LoadInt32(&t.state) == CLOSED
}

func (t *Tube) Free() int {
	c := t.Cap()
	if c < 0 {
		return -1
	}

	return c - t.Running()
}

// ReleaseTimeOut releases all resources allocated by the tube after the specified timeout duration.
// It prevents any new tasks from being submitted within the timeout period. If the tube is already
// closed or the required goroutines for purging and ticktock are not running, it returns ErrTubeClosed.
// It waits for all running goroutines to finish and ensure purging and ticktock completion before
// returning, or returns ErrTimeout if the operation takes longer than the specified timeout.
func (t *Tube) ReleaseTimeOut(timeout time.Duration) error {
	if t.IsClosed() || (!t.options.DisablePurge && t.stopPurge == nil) || t.stopTicktock == nil {
		return ErrTubeClosed
	}

	t.Release()

	var purgeCh <- chan struct{}

	if !t.options.DisablePurge {
		purgeCh = t.purgeCtx.Done()
	} else {
		purgeCh = t.allDone
	}

	if t.Running() == 0 {
		t.once.Do(func(){
			close(t.allDone)
		})
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return ErrTimeout
		case <-t.allDone:
			<-purgeCh
			<-t.ticktockCtx.Done()
			if t.Running() == 0 &&
				(t.options.DisablePurge || atomic.LoadInt32(&t.purgeDone) == 1) &&
				atomic.LoadInt32(&t.ticktockDone) == 1 {
				return nil
			}
		}
	}

}

func(t *Tube) Waiting() int {
	return int(atomic.LoadInt32(&t.waiting))
}

// purgeStaleWorkers periodically purges stale workers from the tube. It starts a goroutine to do so
// and sets up a context with a cancel function to manage the purging process. If the tube is
// closed or the context is canceled, the purging process is stopped. The purging process is
// stopped when all goroutines have finished and the context is canceled. The purging process
// also stops when the tube is closed and there are no running goroutines. If the tube is
// dormant (i.e. there are no running goroutines) and there are waiting tasks, the waiting
// tasks are woken up.
func (t *Tube) purgeStaleWorkers() {
	ticker := time.NewTicker(t.options.ExpiryDuration)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&t.purgeDone, 1)
	}()

	purgeCtx := t.purgeCtx
	for {
		select {
		case <-purgeCtx.Done():
			return
		case <-ticker.C:
		}

		if t.IsClosed() {
			break
		}

		var isDormant bool
		t.lock.Lock()
		staleWorkers := t.workers.refresh(t.options.ExpiryDuration)
		n := t.Running()
		isDormant = n == 0 || n == len(staleWorkers)
		t.now.Store(time.Now())
		t.lock.Unlock()

		for i := range staleWorkers {
			staleWorkers[i].finish()
			staleWorkers[i] = nil
		}

		if isDormant && t.Waiting() > 0 {
			t.cond.Broadcast()
		}
	}
}
// goPurge initializes and starts a goroutine to periodically purge stale workers
// from the tube if purging is not disabled. It sets up a context with a cancel
// function to manage the purging process.
func (t *Tube) goPurge() {
	if !t.options.DisablePurge {
		return
	}

	t.purgeCtx, t.stopPurge = context.WithCancel(context.Background())
	go t.purgeStaleWorkers()
}

func (t *Tube) ticktock() {
	ticker := time.NewTicker(nowTimeUpdateInterval)
	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&t.ticktockDone, 1)
	}()

	ticktockCtx := t.ticktockCtx
	for {
		select {
			case <-ticktockCtx.Done():
				return
			case <-ticker.C:
			
		}

		if t.IsClosed() {
			break
		}
		t.now.Store(time.Now())
	}
}

func(t *Tube) goTicktock() {
	t.now.Store(time.Now())
	t.ticktockCtx, t.stopTicktock = context.WithCancel(context.Background())
	go t.ticktock()
}
func (t *Tube) Reboot() {
	if atomic.CompareAndSwapInt32(&t.state, CLOSED, OPENED) {
		atomic.StoreInt32(&t.purgeDone, 0)
		t.goPurge()
		atomic.StoreInt32(&t.ticktockDone, 0)
		t.goTicktock()
		t.allDone = make(chan struct{})
		t.once = &sync.Once{}

	}
}

func (t *Tube) addRunning(delta int) int {
	return int(atomic.AddInt32(&t.running, int32(delta)))
}

// revertWorker puts a worker back into free tube, recycling goroutines.
func (t *Tube) revertWorker(worker worker) bool {
	if capacity:= t.Cap(); (capacity > 0 && t.Running() > capacity) || t.IsClosed() {
		t.cond.Broadcast()
		return false
	}

	worker.setLastUsedTime(t.nowTime())

	t.lock.Lock()

	if t.IsClosed() {
		t.lock.Unlock()
		return false
	}
	
	if err := t.workers.insert(worker); err != nil {
		t.lock.Unlock()
		return false
	}

	t.cond.Signal()
	t.lock.Unlock()

	return true
}