package commands

import (
	"context"
	"net"
)

func setCustomResolver(dialer *net.Dialer, ip net.IP) {
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	if dialer.Resolver == nil {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
		}
	}

	dialer.Resolver.Dial = func(ctx context.Context, network, address string) (conn net.Conn, e error) {
		d := net.Dialer{}
		return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), `53`))
	}
}
