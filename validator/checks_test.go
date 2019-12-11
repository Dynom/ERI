package validator

import (
	"context"
	"testing"
	"time"
)

func Test_getEarliestDeadlineCTX(t *testing.T) {
	/*
	 * - Test: argument has no deadline
	 * - Test: argument has later deadline
	 * - Test: argument has earlier deadline
	 */

	t.Run("no deadline", func(t *testing.T) {
		ctx, cancel := getEarliestDeadlineCTX(context.Background(), time.Second*10)

		// Timeout shouldn't have expired yet
		if err := ctx.Err(); err != nil {
			t.Errorf("Got error, wasn't expecting that: %+v", err)
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Errorf("Expected a deadline to be set, but it wasn't %+v", deadline)
		}

		cancel()
	})

	t.Run("later parent deadline", func(t *testing.T) {
		parentCTX, parentCancel := context.WithTimeout(context.Background(), time.Second*10)
		parentDeadline, _ := parentCTX.Deadline()

		ctx, cancel := getEarliestDeadlineCTX(parentCTX, time.Second*1)

		// Timeout shouldn't have expired yet
		if err := ctx.Err(); err != nil {
			t.Errorf("Got error, wasn't expecting that: %+v", err)
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Errorf("Expected a deadline to be set, but it wasn't %+v", deadline)
		}

		if !deadline.Before(parentDeadline) {
			t.Errorf("Expected the resulting context to have a deadline before the parent context\nParent: %+v\nResult : %+v", parentDeadline, deadline)
		}

		parentCancel()
		cancel()
	})

	t.Run("earlier parent deadline", func(t *testing.T) {
		parentCTX, parentCancel := context.WithTimeout(context.Background(), time.Second*1)
		parentDeadline, _ := parentCTX.Deadline()

		ctx, cancel := getEarliestDeadlineCTX(parentCTX, time.Second*10)

		// Timeout shouldn't have expired yet
		if err := ctx.Err(); err != nil {
			t.Errorf("Got error, wasn't expecting that: %+v", err)
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Errorf("Expected a deadline to be set, but it wasn't %+v", deadline)
		}

		if !parentDeadline.Equal(deadline) {
			t.Errorf("Expected the resulting context to have the deadline of the parent context\nParent: %+v\nResult : %+v", parentDeadline, deadline)
		}

		parentCancel()
		cancel()
	})

	t.Run("cancel of outer, cancels the inner", func(t *testing.T) {

		parentCTX, parentCancel := context.WithTimeout(context.Background(), time.Second*1)
		ctx, cancel := getEarliestDeadlineCTX(parentCTX, time.Second*10)

		// Timeout shouldn't have expired yet
		if err := ctx.Err(); err != nil {
			t.Errorf("Got error, wasn't expecting that: %+v", err)
		}

		parentCancel()

		// Inner context should also be canceled
		if err := ctx.Err(); err == nil {
			t.Errorf("Expected the inner context to also be cancelled, but it isn't: %+v", ctx)
		}

		cancel()
	})
}
