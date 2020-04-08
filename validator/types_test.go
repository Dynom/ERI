package validator

import (
	"reflect"
	"testing"
)

func Test_createResult(t *testing.T) {
	type args struct {
		a Artifact
	}
	tests := []struct {
		name string
		args args
		want Result
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createResult(tt.args.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
