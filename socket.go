package main

import (
	"dns-resolver/cache"
	"golang.org/x/net/dns/dnsmessage"
	"log"
	"net"
	"sync"
)

type (
	Socket struct {
		args       *ReflectorArgs
		mu         sync.Mutex
		cache      *cache.LRU[dnsmessage.Question, *dnsmessage.Resource]
		parserPoll sync.Pool
		connPoll   sync.Pool
		listener   net.Listener
	}
)

func NewSocket(args *ReflectorArgs) (*Socket, error) {
	listen, err := net.Listen(args.Network, args.Addr)
	if err != nil {
		return nil, err
	}
	log.Printf("started listening on: %s\n", args.Addr)
	lru, err := cache.NewLRU[dnsmessage.Question, *dnsmessage.Resource](args.CacheSize, nil)
	if err != nil {
		return nil, err
	}

	return &Socket{
		args:  args,
		mu:    sync.Mutex{},
		cache: lru,
		parserPoll: sync.Pool{
			New: func() any {
				return dnsmessage.Parser{}
			},
		},
		connPoll: sync.Pool{
			New: func() any {
				c, err := net.Dial(args.Network, args.DNSAddr)
				if err != nil {
					log.Println(err)
					return nil
				}
				return c
			},
		},
		listener: listen,
	}, nil
}

func (s *Socket) HandleAll() error {
	for {
		userConn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handler(userConn)
	}
}

func (s *Socket) handler(userconn net.Conn) {
	parser := s.parserPoll.Get().(dnsmessage.Parser)
	in := make([]byte, 512)
	if _, err := userconn.Read(in); err != nil {
		log.Println(err)
		return
	}
	_, err := parser.Start(in)
	if err != nil {
		log.Println(err)
		return
	}
	//head.
}
