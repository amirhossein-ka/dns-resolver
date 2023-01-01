package main

import (
	"context"
	"dns-resolver/args"
	"dns-resolver/socket"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// set logs output to stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cmdArgs := args.Args{}
	if err := cmdArgs.Parse(); err != nil {
		log.Fatal(err)
	}

	if cmdArgs.OpenConn {
		s, err := socket.NewSocket(cmdArgs.SocketArgs)
		if err != nil {
			log.Fatal(err)
		}
		s.ListenAndServe()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-c
		// s.Stop()
		return

	} else {
		resolver := resolv{
			resolver: &net.Resolver{
				PreferGo:     true,
				StrictErrors: false,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: time.Millisecond * 1000}
					return d.DialContext(ctx, network, net.JoinHostPort(cmdArgs.CmdArgs.DNSAddr, "53"))
				},
			},
			args: &cmdArgs.CmdArgs,
		}
		ch := make(chan Response, len(cmdArgs.CmdArgs.Hosts))

		for _, host := range cmdArgs.CmdArgs.Hosts {
			go resolver.cmdResolve(ch, host)
		}

		for i := 0; i < len(cmdArgs.CmdArgs.Hosts); i++ {
			fmt.Println(<-ch)
		}
	}
}
