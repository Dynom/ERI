package gcp

import (
	"reflect"
	"testing"
)

func TestWithSubscriptionConcurrencyCount(t *testing.T) {
	type args struct {
		c int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "simple test", args: args{c: 10}, want: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := PubSubSvc{
				subscriptionNumProcs: -10, // initial bad value
			}

			WithSubscriptionConcurrencyCount(tt.args.c)(&svc)

			if svc.subscriptionNumProcs != tt.want {
				t.Errorf("WithSubscriptionConcurrencyCount() = %v, want %v", svc.subscriptionNumProcs, tt.want)
			}
		})
	}
}

func TestWithSubscriptionLabels(t *testing.T) {
	type args struct {
		labels []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "simple test", args: args{labels: []string{"a"}}, want: []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := PubSubSvc{
				subscriptionLabels: []string{"foo"},
			}

			WithSubscriptionLabels(tt.args.labels)(&svc)

			if !reflect.DeepEqual(svc.subscriptionLabels, tt.want) {
				t.Errorf("WithSubscriptionConcurrencyCount() = %v, want %v", svc.subscriptionLabels, tt.want)
			}
		})
	}
}
