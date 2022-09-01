package testutil

import (
	"context"
)

func NewContext(parent context.Context) *Context {
	return &Context{
		Context: parent,
	}
}

type Context struct {
	context.Context
	errEvalFn ErrEvalFn
}

type ErrEvalFn func(parent context.Context) error

func (c *Context) SetParent(ctx context.Context) *Context {
	c.Context = ctx
	return c
}

// SetErrEval allows you to define a callback that can be used to influence when Err() returns an error
func (c *Context) SetErrEval(fn ErrEvalFn) {
	c.errEvalFn = fn
}

func (c Context) Err() error {
	if c.errEvalFn == nil {
		return c.Context.Err()
	}

	return c.errEvalFn(c.Context)
}
