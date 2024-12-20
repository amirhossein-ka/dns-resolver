package socket_test

import (
	"dns-resolver/args"
	"dns-resolver/socket"
	"net"
	"testing"
	"time"
)

var (
	req = []byte{0x5e, 0xec, 0x1, 0x20, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x6, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x3, 0x63, 0x6f, 0x6d, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x29, 0x10, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0xc, 0x0, 0xa, 0x0, 0x8, 0x54, 0xb7, 0x8c, 0xc5, 0x5, 0xe4, 0x59, 0x1e}
)

func BenchmarkUdpHandler(b *testing.B) {
	b.StopTimer()
	s, err := socket.NewSocket(args.SocketArgs{
		Addr:      ":8000",
		Network:   "udp",
		DNSAddr:   "1.1.1.1:53",
		CacheSize: 128,
	})
	if err != nil {
		b.Fatal(err)
	}
	go func() {
		s.ListenAndServe()
	}()

	time.Sleep(time.Second)
	conn, err := net.Dial("udp", ":8000")
	if err != nil {
		b.Fatal(err)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err = conn.Write(req)
		if err != nil {
			b.Error(err)
		}
		buf := make([]byte, 256)
		_, err = conn.Read(buf)
		if err != nil {
			b.Error(err)
		}
	}
}
