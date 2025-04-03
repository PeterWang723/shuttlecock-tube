package shuttlecocktube_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	shuttlecock "github.com/PeterWang723/shuttlecock-tube"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
)

const (
	Param    = 100
	AntsSize = 1000
	TestSize = 10000
	n        = 1000000
)

var curMem uint64

const (
	RunTimes           = 1e6
	PoolCap            = 5e4
	BenchParam         = 10
	DefaultExpiredTime = 10 * time.Second
)

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

func TestWaitToGetWorker(t *testing.T){

}

func TestShuttleTube(t *testing.T) {
	defer shuttlecock.Release()
	var wg sync.WaitGroup
	for i:=0; i < n; i++ {
		wg.Add(1)
		_ = shuttlecock.Submit(func() {
			demoFunc()
			wg.Done()
		})
	}

	wg.Wait()

	t.Logf("pool, capacity:%d", shuttlecock.Cap())
	t.Logf("pool, running workers number:%d", shuttlecock.Running())
	t.Logf("pool, free workers number:%d", shuttlecock.Free())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}