package main

import (
	"context"
	args2 "dns-resolver/args"
	"dns-resolver/cache"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Reflector struct {
	args   *args2.ReflectorArgs
	server *dns.Server
	client *dns.Client
	mu     sync.Mutex
	cache  *cache.LRU[key, []dns.RR]
}

type key struct {
	host     string
	respType uint16
}

func newCache(size int) (*cache.LRU[key, []dns.RR], error) {
	c, err := cache.NewLRU[key, []dns.RR](size, nil)
	if err != nil {
		return nil, err
	}

	return c, err
}

func NewReflector(args *args2.ReflectorArgs) *Reflector {
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
		var (
			err error
			res *dns.Msg
		)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		for res == nil {
			res, _, err = r.client.ExchangeContext(ctx, req, r.args.DNSAddr)
			if err != nil {
				log.Println(err)
				return
			}
		}

		res.SetReply(req)
		if err := w.WriteMsg(res); err != nil {
			log.Println(err)
			return
		}

		evicted := r.cache.Add(
			key{respType: res.Question[0].Qtype, host: res.Question[0].Name},
			res.Answer,
		)
		println(evicted)

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

func (r *Reflector) Shutdown() error {
	return r.server.Shutdown()
}
