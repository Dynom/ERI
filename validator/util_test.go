package validator

import (
	"context"
	"net"
	"reflect"
	"testing"
)

func Test_fetchMXHosts(t *testing.T) {

	t.Skipf("figure out how to intercept DNS requests")

	res := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (conn net.Conn, e error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "udp", net.JoinHostPort("localhost", "53"))
		},
	}

	type args struct {
		resolver *net.Resolver
		domain   string
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{wantErr: false, name: "foo", want: []string{"example.org"}, args: args{domain: "foo.bar", resolver: res}},
	}

	ctx := context.Background()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			got, err := fetchMXHosts(ctx, res, tt.args.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchMXHosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchMXHosts() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getConnection(t *testing.T) {
	t.Skipf("figure out how to intercept DNS requests")

	type args struct {
		ctx    context.Context
		dialer *net.Dialer
		mxHost string
	}
	tests := []struct {
		name    string
		args    args
		want    net.Conn
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getConnection(tt.args.ctx, tt.args.dialer, tt.args.mxHost)
			if (err != nil) != tt.wantErr {
				t.Errorf("getConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getConnection() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_MightBeAHostOrIP(t *testing.T) {
	type args struct {
		h string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{want: true, name: "IP", args: args{h: "127.0.0.1"}},

		{want: true, name: "Domain common", args: args{h: "example.org"}},
		{want: true, name: "Domain multi-tld", args: args{h: "example.co.uk"}},
		{want: true, name: "Domain CaSed", args: args{h: "eXample.Org"}},
		{want: true, name: "Domain dash", args: args{h: "ex-ample.org"}},

		{name: "dot", args: args{h: "."}},
		{name: "bad, suffix dot", args: args{h: ".example.org"}},
		{name: "bad, postfix dot", args: args{h: "example.org."}},
		{name: "bad, no tld", args: args{h: "exampleorg"}},
		{name: "bad, bad char", args: args{h: "example!org"}},
		{name: "bad, space", args: args{h: " example.org"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MightBeAHostOrIP(tt.args.h); got != tt.want {
				t.Errorf("MightBeAHostOrIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
