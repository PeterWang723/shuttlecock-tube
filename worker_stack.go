package shuttlecocktube

import (
	"time"
)

type workerStack struct {
	items []worker
	expiry []worker
}

func newWorkerStack(size int) *workerStack {
	return &workerStack{
		items: make([]worker, 0, size),
	}
}

func (s *workerStack) len() int {
	return len(s.items)
}

func(s *workerStack) insert(w worker) error {
	s.items = append(s.items, w)
	return nil
}

func (s *workerStack) isEmpty() bool {
	return len(s.items) == 0
}

func (s *workerStack) detach() worker {
	l := s.len()
	if l == 0 {
		return nil
	}

	w := s.items[l-1]
	s.items[l-1] = nil
	s.items = s.items[:l-1]
	return w
}

func (s *workerStack) refresh(duration time.Duration) []worker {
	n:= s.len()
	if n == 0{
		return nil
	}

	expiryTime := time.Now().Add(-duration)
	index := s.binarySearch(0, n-1, expiryTime)

	s.expiry = s.expiry[:0]
	if index != -1 {
		s.expiry = append(s.expiry, s.items[:index+1]...)
		m := copy(s.items, s.items[index+1:])
		for i:=m; i<n; i++ {
			s.items[i] = nil
		}
		s.items = s.items[:m]
	}
	return s.expiry
}

func (s *workerStack) binarySearch(l, r int, expiryTime time.Time) int {
	for l <= r {
		mid := l+((r-l) >> 1)
		if expiryTime.Before(s.items[mid].lastUsedTime()) {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}
	return r
}

func (s *workerStack) reset() {
	for i := 0; i < s.len(); i++ {
		s.items[i].finish()
		s.items[i] = nil
	}

	s.items = s.items[:0]
}