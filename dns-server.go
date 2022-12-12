package main

import (
	"context"
	"dns-resolver/cache"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

type Reflector struct {
	args   *ReflectorArgs
	server *dns.Server
	client *dns.Client
	cache  *cache.LRU[key, []dns.RR]
}

type key struct {
	host     string
	respType uint16
}

func newCache(size int) (*cache.LRU[key, []dns.RR], error) {
	onEvict := func(key key, val []dns.RR) {
		if key.host != val[0].Header().Name {
			log.Println("key and value does not match")
		}
	}

	c, err := cache.NewLRU(size, onEvict)
	if err != nil {
		return nil, err
	}

	return c, err
}

func NewReflector(args *ReflectorArgs) *Reflector {
	s := &dns.Server{Addr: args.Addr, Net: args.Network, TsigSecret: nil, ReusePort: true}
	c := new(dns.Client)
	c.Dialer = &net.Dialer{
		Timeout: 300 * time.Millisecond,
	}
	lru, err := newCache(args.CacheSize)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return &Reflector{
		args:   args,
		server: s,
		client: c,
		cache:  lru,
	}
}

func (r *Reflector) handleReflect(w dns.ResponseWriter, req *dns.Msg) {
	// if the host already exists on cache and type of requested record is available
	q := req.Question[0]
	if ans, ok := r.cache.Get(key{respType: q.Qtype, host: q.Name}); ok {
		fmt.Println("response from cache")
		m := new(dns.Msg)
		m.SetReply(req)
		m.Question[0] = req.Question[0]
		m.Answer = ans
		if err := w.WriteMsg(m); err != nil {
			log.Println(err)
			return
		}
	} else {
		// retry exchange
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*3000)
			res, _, err := r.client.ExchangeContext(ctx, req, r.args.DNSAddr)
			cancel()
			if err != nil {
				log.Println(err)
				continue
			}
			res.SetReply(req)
			r.cache.Add(
				key{respType: res.Question[0].Qtype, host: res.Question[0].Name},
				res.Answer,
			)

			if err := w.WriteMsg(res); err != nil {
				log.Println(err)
				continue
			}
		}
	}

}

func (r *Reflector) Serve() error {
	dns.HandleFunc(".", r.handleReflect)
	log.Println("starting server")
	if err := r.server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
