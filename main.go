package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	// set logs output to stderr
	log.SetOutput(os.Stderr)
	args := Args{}
	if err := args.parse(); err != nil {
		log.Fatal(err)
	}

	resolver := resolv{
		resolver: &net.Resolver{
			PreferGo:     true,
			StrictErrors: false,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: time.Millisecond * 1000}
				return d.DialContext(ctx, network, net.JoinHostPort(args.CmdArgs.DNSAddr, "53"))
			},
		},
		args: &args.CmdArgs,
	}

	if args.OpenConn {
		if err := NewReflector(&args.SocketArgs).Serve(); err != nil {
			log.Fatal(err)
		}

	} else {

		ch := make(chan Response, len(args.CmdArgs.Hosts))

		for _, host := range args.CmdArgs.Hosts {
			go resolver.cmdResolve(ch, host)
		}

		for i := 0; i < len(args.CmdArgs.Hosts); i++ {
			fmt.Println(<-ch)
		}
	}
}
