package testutil

import (
	"context"
	"errors"
	"testing"
)

func TestContext_Err(t *testing.T) {
	ctx := context.Background()
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	type fields struct {
		Context   context.Context
		errEvalFn ErrEvalFn
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "nil err, when no err and no errFn",
			wantErr: false,
			fields: fields{
				Context:   ctx,
				errEvalFn: nil,
			},
		},
		{
			name:    "nil err, when no err and no erroneous errFn",
			wantErr: false,
			fields: fields{
				Context: ctx,
				errEvalFn: func(parent context.Context) error {
					return nil
				},
			},
		},
		{
			name:    "err, when context err and no errFn",
			wantErr: true,
			fields: fields{
				Context:   canceledCtx,
				errEvalFn: nil,
			},
		},
		{
			name:    "err, when no err and erroneous errFn",
			wantErr: true,
			fields: fields{
				Context: ctx,
				errEvalFn: func(parent context.Context) error {
					return errors.New("foo")
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := NewContext(tt.fields.Context)
			c.SetErrEval(tt.fields.errEvalFn)

			if err := c.Err(); (err != nil) != tt.wantErr {
				t.Errorf("Err() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContext_SetParent(t *testing.T) {
	c := NewContext(nil)
	if c.Context != nil {
		t.Errorf("Expected a nil context, instead it was %#v", c.Context)
	}

	c.SetParent(context.Background())
	if c.Context == nil {
		t.Errorf("Expected a non-nil context, instead it was %#v", c.Context)
	}
}
