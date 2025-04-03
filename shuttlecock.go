package shuttlecocktube

import (
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)


const (
	// default tube size is the default size of the goroutine pool
	DefaultTubeSize = math.MaxInt32

	// default clean interval is the defualt time for cleaning up goroutines
	DefaultCleanInterval = time.Second
)

const (
	// OPENED and CLOSED are the states of the tube
	OPENED = iota

	CLOSED
)

var (
	// ErrLackTubeFunction is the error when there is no tube function
	ErrLackTubeFunction = errors.New("lack of tube function")

	// ErrIvalidTubeExpiry is the error when the tube expiry is invalid
	ErrIvalidTubeExpiry = errors.New("invalid tube expiry")

	// ErrTubeClosed is the error when the tube is closed
	ErrTubeClosed = errors.New("tube is closed")

	// ErrTubeOverload is the error when the tube is overloaded
	ErrTubeOverload = errors.New("tube is blocked or nonblocking is set")

	// ErrInvalidPreAllocSize is the error when the prealloc size is invalid
	ErrInvalidPreAllocSize = errors.New("invalid prealloc size")

	// ErrTimeout is the error when the operation timeout
	ErrTimeout = errors.New("timeout")

	// ErrInvalidTubeIndex is the error when the tube index is invalid
	ErrInvalidTubeIndex = errors.New("invalid tube index")

	// ErrInvalidLoadBalancingStrategy is the error when the load balancing strategy is invalid
	ErrInvalidLoadBalancingStrategy = errors.New("invalid load balancing strategy")
	
	// workerChanCap is the capacity of the worker channel 
	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}()

	// defaultLogger is the default logger
	defaultLogger = Logger(log.New(os.Stderr, "[shuttlecock]: ", log.LstdFlags|log.Lmsgprefix|log.Lmicroseconds))
	
	// defaultShuttleTube is the default shuttlecock tube
	defaultShuttleTube, _ = NewTube(DefaultTubeSize)
)

// nowTimeUpdateInterval is the interval for updating the current time
const nowTimeUpdateInterval = 500 * time.Millisecond

// Submit submits a task to the default shuttlecock tube for execution.
// Returns an error if the task cannot be submitted.
func Submit(task func()) error {
	return defaultShuttleTube.Submit(task)
}

// Running returns the number of currently running goroutines in the default shuttlecock tube.
func Running() int {
	return defaultShuttleTube.Running()
}

// Cap returns the current capacity of the default shuttlecock tube.
func Cap() int {
	return defaultShuttleTube.Cap()
}

// Free returns the number of available goroutine slots in the default shuttlecock tube.
func Free() int {	
	return defaultShuttleTube.Free()
}

// Release releases all resources allocated by the default shuttlecock tube and
// prevents any new tasks from being submitted. It does not wait for any
// currently running goroutines to finish.
func Release() {
	defaultShuttleTube.Release()
}

// ReleaseTimeOut releases all resources allocated by the default shuttlecock tube after the specified timeout duration.
// It prevents any new tasks from being submitted within the timeout period.
// Returns an error if the release operation times out.
func ReleaseTimeOut(timeout time.Duration) error {
	return defaultShuttleTube.ReleaseTimeOut(timeout)
}

// Reboot reinitializes the default shuttlecock tube, resetting its state and resources.
// This function can be used to restart the tube after releasing it, allowing new tasks
// to be submitted and processed.
func Reboot() {
	defaultShuttleTube.Reboot()
}