package args

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

type (
	Hosts []string
	Args  struct {
		OpenConn   bool
		CmdArgs    CmdArgs
		SocketArgs ReflectorArgs
		Redis      Redis
	}
	CmdArgs struct {
		Hosts              Hosts
		A, AAAA, MX, CNAME bool
		DNSAddr            string
		resolver           *net.Resolver
	}

	ReflectorArgs struct {
		Addr      string
		Network   string
		DNSAddr   string
		CacheSize int
	}

	Redis struct {
		Addr     string
		Password string
		DB       int
	}
)

// String return the default host for query
func (h *Hosts) String() string {
	return "google.com"
}

func (h *Hosts) Set(val string) error {
	*h = append(*h, val)
	return nil
}

func (a *Args) Parse() error {
	cmd := flag.NewFlagSet("cmd", flag.ExitOnError)
	cmd.BoolVar(&a.CmdArgs.A, "a", true, "search for A record")
	cmd.BoolVar(&a.CmdArgs.AAAA, "aaaa", false, "search for AAAA record")
	cmd.BoolVar(&a.CmdArgs.CNAME, "cname", false, "search for CNAME record")
	cmd.BoolVar(&a.CmdArgs.MX, "mx", false, "search for MX record")
	cmd.StringVar(&a.CmdArgs.DNSAddr, "dns", "1.1.1.1:53", "set custom dns for resolver")
	cmd.Var(&a.CmdArgs.Hosts, "host", "hosts to get their ip (can be used mutiple times)")

	server := flag.NewFlagSet("server", flag.ExitOnError)
	server.StringVar(&a.SocketArgs.Addr, "addr", ":8000", "addr to listen on it")
	server.StringVar(&a.SocketArgs.Network, "net", "udp", "socket type")
	server.StringVar(&a.SocketArgs.DNSAddr, "dns", "1.1.1.1:53", "set custom dns for resolver")
	server.IntVar(&a.SocketArgs.CacheSize, "cachesize", 128, "cache size list")

	if len(os.Args) < 2 {
		return fmt.Errorf("error occured while parsing flags: expected 'cmd' or 'server' subcommands")
	}

	switch os.Args[1] {
	case "cmd":
		if err := cmd.Parse(os.Args[2:]); err != nil {
			return err
		}
	case "server":
		if err := server.Parse(os.Args[2:]); err != nil {
			return err
		}
		a.OpenConn = true
	default:
		return fmt.Errorf("error occured while parsing flags: expected 'cmd' or 'server' subcommands")
	}

	a.CmdArgs.resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * 1000,
			}
			return d.DialContext(ctx, network, net.JoinHostPort(a.CmdArgs.DNSAddr, "53"))
		},
	}

	return nil
}
