package testutil

import (
	"context"
	"time"
)

func NewContext(parent context.Context) *Context {
	return &Context{
		parent: parent,
	}
}

type Context struct {
	parent    context.Context
	errEvalFn ErrEvalFn

	//context.Context
}

type ErrEvalFn func(parent context.Context) error

func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.parent.Deadline()
}

func (c Context) Done() <-chan struct{} {
	return c.parent.Done()
}

func (c *Context) SetParent(ctx context.Context) *Context {
	c.parent = ctx
	return c
}

// SetErrEval allows you to define a callback that can be use to influence when Err() returns an error
func (c *Context) SetErrEval(fn ErrEvalFn) {
	c.errEvalFn = fn
}

func (c Context) Err() error {
	if c.errEvalFn == nil {
		return c.parent.Err()
	}

	return c.errEvalFn(c.parent)
}

func (c Context) Value(key interface{}) interface{} {
	return c.parent.Value(key)
}
