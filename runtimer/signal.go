package runtimer

import (
	"os"
	"os/signal"
	"sync"
)

type Callback func(s os.Signal)

func New(signals ...os.Signal) *SignalHandler {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)

	sh := &SignalHandler{
		c:    c,
		done: make(chan struct{}),
		lock: sync.Mutex{},
	}

	go sh.handle()

	return sh
}

type SignalHandler struct {
	c    chan os.Signal
	done chan struct{}
	fns  []Callback
	lock sync.Mutex
}

func (sh *SignalHandler) handle() {
	defer func() {
		sh.done <- struct{}{}
	}()

	s := <-sh.c
	signal.Stop(sh.c)
	close(sh.c)

	for _, fn := range sh.fns {
		fn(s)
	}
}

func (sh *SignalHandler) RegisterCallback(fn Callback) {
	sh.lock.Lock()
	sh.fns = append(sh.fns, fn)
	sh.lock.Unlock()
}

// Wait block until all callback's have been called
func (sh *SignalHandler) Wait() {
	<-sh.done
	close(sh.done)
}
