package main

import (
	"dns-resolver/cache"
	"fmt"
	"log"
	"net"
	"sync"

	"golang.org/x/net/dns/dnsmessage"
)

type (
	Socket struct {
		args       *ReflectorArgs
		mu         sync.Mutex
		cache      *cache.LRU[dnsmessage.Question, []dnsmessage.Resource]
		parserPoll sync.Pool
		connPoll   sync.Pool
		listener   net.PacketConn
	}

	socketKey struct {
		name string
		Type uint16
	}
)

func NewSocket(args *ReflectorArgs) (*Socket, error) {
	var (
		listen net.PacketConn
		err    error
	)
	if args.Network == "udp" {
		listen, err = net.ListenPacket(args.Network, args.Addr)
	} else {
		return nil, fmt.Errorf("network not supported")

	}
	if err != nil {
		return nil, err
	}
	log.Printf("started listening on: %s\n", args.Addr)
	lru, err := cache.NewLRU[dnsmessage.Question, []dnsmessage.Resource](args.CacheSize, nil)
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

func (s *Socket) Serve() {
	for {
		buf := make([]byte, 1024)
		readLen, addr, err := s.listener.ReadFrom(buf)
		if err != nil {
			log.Println(err)
			return
		}

		go s.udpHandler(addr, buf[:readLen])
	}
}

func (s *Socket) udpHandler(addr net.Addr, in []byte) {
	parser, ok := s.parserPoll.Get().(dnsmessage.Parser)
	if !ok {
		log.Println("cant get parser")
		return
	}

	header, err := parser.Start(in)
	if err != nil {
		log.Println(err)
		return
	}

	question, err := parser.Question()
	if err != nil {
		log.Println(err)
		return
	}

	if answer, ok := s.cache.Get(question); ok {
		response := new(dnsmessage.Message)
		response.Questions = append(response.Questions, question)
		response.Header = header
		response.Answers = answer

		msg, err := response.Pack()
		if err != nil {
			log.Println(err)
			return
		}

		_, err = s.listener.WriteTo(msg, addr)
		if err != nil {
			log.Println(err)
			return
		}

	} else {
		parser2 := s.parserPoll.Get().(dnsmessage.Parser)
		remoteDns, ok := s.connPoll.Get().(net.Conn)
		defer remoteDns.Close()
		if !ok {
			return
		}
		// redirect the query to remoteDns
		_, err := remoteDns.Write(in)
		if err != nil {
			log.Println(err)
			return
		}

		resp := make([]byte, 514)

		_, err = remoteDns.Read(resp)
		if err != nil {
			log.Println(err)
			return
		}

		_, err = s.listener.WriteTo(resp, addr)
		if err != nil {
			log.Println(err)
			return
		}

		_, err = parser2.Start(resp)
		if err != nil {
			log.Println(err)
			return
		}

		q, err := parser2.AllQuestions()
		if err != nil {
			log.Println(err)
			return
		}
		r, err := parser2.Answer()
		if err != nil {
			log.Println(err)
			return
		}

		s.cache.Add(q[0], []dnsmessage.Resource{r})
	}
}
