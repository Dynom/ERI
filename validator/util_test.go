package validator

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
)

type stubDialer struct {
	err error
}

func (sd stubDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// We don't use net.Conn, so returning nil for the interface here is safe
	return nil, sd.err
}

type stubResolver struct {
	mxs []*net.MX
	err error // A single error for every resolver
}

func (sr stubResolver) LookupMX(_ context.Context, domain string) ([]*net.MX, error) {
	return sr.mxs, sr.err
}

func buildLookupMX(mxHosts []string, err error) LookupMX {
	var r stubResolver
	r.err = err

	r.mxs = make([]*net.MX, len(mxHosts))
	for i, d := range mxHosts {
		r.mxs[i] = &net.MX{
			Host: d,
			Pref: uint16(i),
		}
	}
	return r
}

func Test_fetchMXHosts(t *testing.T) {

	type args struct {
		hosts []string
		err   error
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{

		// The good
		{name: "Happy flow", want: []string{"mx1.example.org"}, args: args{hosts: []string{"mx1.example.org"}}},

		// The bad
		{wantErr: true, name: "no MX records", want: []string{}, args: args{hosts: []string{}}},
		{wantErr: true, name: "malformed MX records", want: []string{}, args: args{hosts: []string{"."}}},
		{wantErr: true, name: "lookup error", want: []string{}, args: args{hosts: []string{"."}, err: errors.New("err")}},
	}

	ctx := context.Background()
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchMXHosts(ctx, buildLookupMX(tt.args.hosts, tt.args.err), "foobar.local")

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

	// @todo validate connection timeout workings
	// @todo

	type args struct {
		err error
	}

	tests := []struct {
		name    string
		args    args
		want    net.Conn
		wantErr bool
	}{
		// The good
		{name: "happy flow", args: args{err: nil}},
		{name: "expected error", args: args{err: errors.New("connection refused")}},

		// The bad
		{wantErr: true, name: "unexpected error", args: args{err: errors.New("b0rk")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dialer := stubDialer{}
			dialer.err = tt.args.err

			got, err := getConnection(ctx, dialer, "mx.example.org")
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
		{want: true, name: "postfix dot", args: args{h: "example.org."}},

		{name: "dot", args: args{h: "."}},
		{name: "bad, suffix dot", args: args{h: ".example.org"}},
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
