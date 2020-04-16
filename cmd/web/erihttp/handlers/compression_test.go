package handlers

import (
	"reflect"
	"testing"
)

func TestWithGzipHandler(t *testing.T) {
	tests := []struct {
		name string
		want HandlerWrapper
	}{
		{
			name: "test",
			want: WithGzipHandler(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithGzipHandler(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithGzipHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
