package validator

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/Dynom/ERI/types"
)

func newStubDialer(err error) *stubDialer {
	return &stubDialer{
		conn: &net.IPConn{},
		err:  err,
	}
}

type stubDialer struct {
	err  error
	conn net.Conn
}

func (sd stubDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// We don't use net.Conn, so returning nil for the interface here is safe
	return sd.conn, sd.err
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
		{wantErr: true, name: "no MX records", want: nil, args: args{hosts: []string{}}},
		{wantErr: true, name: "lookup error", want: nil, args: args{hosts: []string{"."}, err: errors.New("err")}},

		// We had a result, but all were invalid. The result is an empty slice instead of a nil slice.
		{wantErr: true, name: "malformed MX records", want: []string{}, args: args{hosts: []string{"."}}},
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
				t.Errorf("fetchMXHosts() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_getConnection(t *testing.T) {
	// @todo validate connection timeout workings
	// @todo

	type args struct {
		err  error
		conn net.Conn
	}

	defaultConn := &net.IPConn{}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// The good
		{name: "happy flow", args: args{err: nil, conn: defaultConn}},
		{name: "expected error", args: args{err: errors.New("connection refused"), conn: defaultConn}},

		// The bad
		{wantErr: true, name: "unexpected error", args: args{err: errors.New("b0rk"), conn: defaultConn}},
		{wantErr: true, name: "no conn", args: args{err: nil, conn: nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			dialer := newStubDialer(tt.args.err)
			dialer.conn = tt.args.conn

			_, err := getConnection(ctx, dialer, "mx.example.org")
			if (err != nil) != tt.wantErr {
				t.Errorf("getConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestEmailValidator_getNewArtifact(t *testing.T) {
	t.Run("Dialer is set with default resolver", func(t *testing.T) {
		ctx := context.Background()
		a := getNewArtifact(ctx, types.EmailParts{}, WithDialer(&net.Dialer{Resolver: nil}))
		if a.dialer == nil || a.resolver == nil {
			t.Errorf("Expected a default dialer to be used, it didn't %+v", a.dialer)
		}
	})
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
		{want: true, name: "nice and short", args: args{h: "az.de"}},

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

func Test_wrapError(t *testing.T) {
	errA := errors.New("a")
	errB := errors.New("b")

	type args struct {
		parent error
		new    error
	}
	tests := []struct {
		name        string
		args        args
		wantWrapped error
		wantErr     bool
	}{
		{
			name: "is wrapped",
			args: args{
				parent: errA,
				new:    errB,
			},
			wantWrapped: errB,
			wantErr:     true,
		},
		{
			name: "nil parent",
			args: args{
				parent: nil,
				new:    errB,
			},
			wantWrapped: errB,
			wantErr:     true,
		},
		{
			name: "zero-value",
			args: args{
				parent: nil,
				new:    nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapError(tt.args.parent, tt.args.new)

			if !errors.Is(err, tt.wantWrapped) {
				t.Errorf("errors.Is() error %q isn't wrapping %q (err: %v)", tt.args.parent, tt.wantWrapped, err)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("wrapError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
