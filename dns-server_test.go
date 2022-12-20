package main

import (
	"dns-resolver/args"
	"github.com/miekg/dns"
	"testing"
)

var (
	reflector = NewReflector(&args.ReflectorArgs{
		Addr:      ":8053",
		Network:   "udp",
		DNSAddr:   "1.1.1.1:53",
		CacheSize: 16,
	})
)

// not working
func BenchmarkHandler(b *testing.B) {
	b.StopTimer()
	go func() {
		if err := reflector.Serve(); err != nil {
			b.Errorf("error while serving server: %v\n", err)
			return
		}
	}()
	defer reflector.Shutdown()

	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("gmail.com"), dns.TypeMX)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := c.Exchange(m, ":8053"); err != nil {
			b.Fatalf("err: %v\n", err)
		}
	}
	// TODO:
	// check this : https://github.com/miekg/dns/blob/master/server_test.go
	// and this: https://github.com/miekg/dns/blob/master/serve_mux_test.go
	// to benchmark this app
}
