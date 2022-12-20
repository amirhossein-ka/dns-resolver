package main

import (
	"context"
	args2 "dns-resolver/args"
	"fmt"
	"net"
)

type resolv struct {
	resolver *net.Resolver
	args     *args2.CmdArgs
}

func (r *resolv) cmdResolve(ch chan Response, host string) {
	var (
		IPs    []net.IP
		MXs    []*net.MX
		CNAME  string
		err    error
		errors []error
	)
	if r.args.A {
		addr, err := r.resolver.LookupIP(context.TODO(), "ip4", host)
		if err != nil {
			errors = append(errors, fmt.Errorf("get A record: %w", err))
		}

		for i := 0; i < len(addr); i++ {
			IPs = append(IPs, addr[i])
		}
	}
	if r.args.AAAA {
		addr, err := r.resolver.LookupIP(context.TODO(), "ip6", host)
		if err != nil {
			errors = append(errors, fmt.Errorf("get AAAA record: %w", err))
		}

		for i := 0; i < len(addr); i++ {
			IPs = append(IPs, addr[i])
		}

	}
	if r.args.MX {
		MXs, err = r.resolver.LookupMX(context.TODO(), host)
		if err != nil {
			errors = append(errors, fmt.Errorf("get MX record: %w", err))
		}
	}
	if r.args.CNAME {
		CNAME, err = r.resolver.LookupCNAME(context.TODO(), host)
		if err != nil {
			errors = append(errors, fmt.Errorf("get CNAME record: %w", err))
		}
	}

	ch <- Response{
		Host:        host,
		IPs:         IPs,
		MXrecord:    MXs,
		CNAMErecord: CNAME,
		Error:       errors,
	}
}
