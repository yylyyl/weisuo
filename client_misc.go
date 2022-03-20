package main

import (
	"context"
	"log"
	"net"
	"net/url"
)

func getClientResolverDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	if cfg.ClientResolver == "" {
		return nil
	}

	u, err := url.Parse(cfg.ClientResolver)
	if err != nil {
		log.Fatalf("cannot parse value of client_resolver: %v", err)
	}

	switch u.Scheme {
	case "udp":
	case "tcp":
	default:
		log.Fatalf("Supported schemes of client_resolver: udp, tcp. Got %s", u.Scheme)
	}

	resolverDialer := &net.Dialer{}

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return resolverDialer.DialContext(ctx, u.Scheme, u.Host)
			},
		},
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
}
