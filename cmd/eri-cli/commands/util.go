package commands

import (
	"context"
	"net"
	"os"
)

// isStdinPiped returns true if our input is from a pipe
func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return isPiped(fi)
}

// isStdinPiped returns true if the output is a pipe
func isStdoutPiped() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return isPiped(fi)
}

func isPiped(fi os.FileInfo) bool {
	if fi == nil {
		return false
	}

	return fi.Mode()&os.ModeNamedPipe == os.ModeNamedPipe
}

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
