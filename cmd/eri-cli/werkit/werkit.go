package werkit

import (
	"context"
	"sync"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
)

type CheckTask struct {
	Ctx   context.Context
	Fn    validator.CheckFn
	Parts types.EmailParts
}

type WerkIt struct {
	wg    sync.WaitGroup
	tasks chan CheckTask
}

func (wi *WerkIt) StartCheckWorkers(workers int, fn func(tasks <-chan CheckTask)) {

	wi.tasks = make(chan CheckTask)
	wi.wg.Add(workers)
	for i := workers; i > 0; i-- {
		go func() {
			defer wi.wg.Done()
			fn(wi.tasks)
		}()
	}
}

func (wi *WerkIt) Wait() {
	close(wi.tasks)
	wi.wg.Wait()
}

func (wi *WerkIt) Process(t CheckTask) {
	wi.tasks <- t
}
