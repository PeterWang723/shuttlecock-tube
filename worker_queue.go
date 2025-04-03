package shuttlecocktube

import (
	"errors"
	"time"
)


var (
	// errQueueIsFull will be returned when the worker queue is full.
	errQueueIsFull = errors.New("the queue is full")

	// errQueueIsReleased will be returned when trying to insert item to a released worker queue.
	errQueueIsReleased = errors.New("the queue length is zero")
)
// worker is the interface for a worker
type worker interface{
	run() 
	finish()
	lastUsedTime() time.Time
	setLastUsedTime(time.Time)
	inputFunc(func())
	inputArg(any)
}

// workerQueue is a interface for a queue for workers
type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker) error
	detach() worker
	refresh(timeout time.Duration) []worker
	reset()
}

type queueType int

const (
	queueTypeStack queueType = iota
	queueTypeLoopQueue
)

func newWorkerQueue(qType queueType, size int) workerQueue {
	switch qType {
		case queueTypeStack:
			return newWorkerStack(size)
		default:
			return newWorkerStack(size)
	}
}

