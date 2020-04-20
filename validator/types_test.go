package validator

import (
	"reflect"
	"testing"

	"github.com/Dynom/ERI/validator/validations"
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
		{
			name: "Testing if result was created",
			args: args{
				a: Artifact{
					Validations: validations.Validations(validations.FValid),
					Steps:       validations.Steps(validations.FValid),
				},
			},
			want: Result{
				Validations: validations.Validations(validations.FValid),
				Steps:       validations.Steps(validations.FValid),
			},
		},
		{
			name: "Testing with zero-input",
			args: args{
				a: Artifact{
					Validations: 0,
					Steps:       0,
				},
			},
			want: Result{
				Validations: 0,
				Steps:       0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createResult(tt.args.a); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
