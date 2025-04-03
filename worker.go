package shuttlecocktube

import (
	"runtime/debug"
	"time"
)

// goWorker is the actual executor who runs the tasks,
// it starts a goroutine that accepts tasks and
// performs function calls.
type goWorker struct {
	// tube who owns this worker.
	tube *Tube

	// task is a job should be done.
	task chan func()

	// lastUsed will be updated when putting a worker back into queue.
	lastUsed time.Time
}

// run starts a goroutine to repeat the process
// that performs the function calls.
func (w *goWorker) run() {
	w.tube.addRunning(1)
	go func() {
		defer func() {
			if w.tube.addRunning(-1) == 0 && w.tube.IsClosed() {
				w.tube.once.Do(func() {
					close(w.tube.allDone)
				})
			}
			w.tube.workerCache.Put(w)
			if p:= recover(); p != nil {
				if ph := w.tube.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.tube.options.Logger.Printf("worker exits from panic: %v\n%s\n", p, debug.Stack())
				}
			}
		}()

		for f := range w.task {
			if f == nil {
				return
			}
			f()
			if ok := w.tube.revertWorker(w); !ok {
				return
			}
		}

	}()
}

func (w *goWorker) finish() {
	w.task <- nil
}

func (w *goWorker) inputFunc(fn func()) {
	w.task <- fn
}

func (w *goWorker) lastUsedTime() time.Time {
	return w.lastUsed
}

func (w *goWorker) setLastUsedTime(t time.Time) {
	w.lastUsed = t
}

func (w *goWorker) inputArg(arg any) {
	return
}