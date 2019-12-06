package validator

import (
	"context"
	"net"
)

type LookupMX interface {
	LookupMX(ctx context.Context, domain string) ([]*net.MX, error)
}

type DialContext interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
