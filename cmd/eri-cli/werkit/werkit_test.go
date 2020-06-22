package werkit

import (
	"context"
	"sync"
	"testing"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
)

func TestWerkItSingleWorker(t *testing.T) {

	const expect = 1
	var testCounter = 0

	wi := &WerkIt{}
	wi.StartCheckWorkers(1, func(tasks <-chan CheckTask) {
		for task := range tasks {
			task.Fn(nil, types.EmailParts{})
		}
	})

	wi.Process(CheckTask{
		Fn: func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
			testCounter++
			return validator.Result{}
		},
	})

	wi.Wait()

	if testCounter != expect {
		t.Errorf("Expected %d runs, instead the counter is %d", expect, testCounter)
	}
}

func TestWerkItManyWorkers(t *testing.T) {
	const expect = 1
	var testCounter = 0

	wi := &WerkIt{}
	wi.StartCheckWorkers(100, func(tasks <-chan CheckTask) {
		for task := range tasks {
			task.Fn(nil, types.EmailParts{})
		}
	})

	var lock = sync.RWMutex{}
	wi.Process(CheckTask{
		Fn: func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
			lock.Lock()
			testCounter++
			lock.Unlock()

			return validator.Result{}
		},
	})

	wi.Wait()

	if testCounter != expect {
		t.Errorf("Expected %d runs, instead the counter is %d", expect, testCounter)
	}
}

func TestWerkItSingleWorkerMultipleRuns(t *testing.T) {

	const expect = 100
	var testCounter = 0

	wi := &WerkIt{}
	wi.StartCheckWorkers(1, func(tasks <-chan CheckTask) {
		for task := range tasks {
			task.Fn(nil, types.EmailParts{})
		}
	})

	var fn = func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
		testCounter++
		return validator.Result{}
	}
	for i := expect; i > 0; i-- {
		wi.Process(CheckTask{
			Fn: fn,
		})
	}

	wi.Wait()

	if testCounter != expect {
		t.Errorf("Expected %d runs, instead the counter is %d", expect, testCounter)
	}
}

func TestWerkItManyWorkerMultipleRuns(t *testing.T) {

	const expect = 10000
	var testCounter = 0

	wi := &WerkIt{}
	wi.StartCheckWorkers(500, func(tasks <-chan CheckTask) {
		for task := range tasks {
			task.Fn(nil, types.EmailParts{})
		}
	})

	var lock = sync.RWMutex{}
	var fn = func(ctx context.Context, parts types.EmailParts, options ...validator.ArtifactFn) validator.Result {
		lock.Lock()
		testCounter++
		lock.Unlock()

		return validator.Result{}
	}

	for i := expect; i > 0; i-- {
		wi.Process(CheckTask{
			Fn: fn,
		})
	}

	wi.Wait()

	if testCounter != expect {
		t.Errorf("Expected %d runs, instead the counter is %d", expect, testCounter)
	}
}
