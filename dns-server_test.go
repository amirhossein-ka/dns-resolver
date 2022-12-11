package main

import "testing"

var (
	reflector = NewReflector(&ReflectorArgs{
		Addr:      ":8053",
		Network:   "udp",
		DNSAddr:   "1.1.1.1:54",
		CacheSize: 16,
	})
)

func BenchmarkReflector_Serve(b *testing.B) {
	b.StopTimer()
	// TODO:
	// check this : https://github.com/miekg/dns/blob/master/server_test.go
	// and this: https://github.com/miekg/dns/blob/master/serve_mux_test.go
}
