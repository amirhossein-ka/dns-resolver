package main

import (
	"dns-resolver/args"
	"dns-resolver/cache"
	"fmt"
	"log"
	"net"
	"sync"

	"golang.org/x/net/dns/dnsmessage"
)

type (
	Socket struct {
		mu       sync.Mutex
		cache    *cache.LRU[dnsmessage.Question, []dnsmessage.Resource]
		bufPopl  sync.Pool
		connPoll sync.Pool
		listener *net.UDPConn
		cancel   chan struct{}
		data     chan data
	}
	data struct {
		response []byte
		addr     *net.UDPAddr
	}
)

func NewSocket(args args.ReflectorArgs) (*Socket, error) {
	var (
		listen *net.UDPConn
		err    error
	)

	localAddr, err := net.ResolveUDPAddr(args.Network, args.Addr)
	if err != nil {
		return nil, err
	}

	if args.Network == "udp" {
		listen, err = net.ListenUDP(args.Network, localAddr)
	} else {
		return nil, fmt.Errorf("network not supported")

	}
	if err != nil {
		return nil, err
	}
	log.Printf("started listening on: %s\n", args.Addr)

	onEvict := func(_ dnsmessage.Question, _ []dnsmessage.Resource) {
	}

	lru, err := cache.NewLRU(args.CacheSize, onEvict)
	if err != nil {
		return nil, err
	}

	return &Socket{
		//args:  args,
		mu:    sync.Mutex{},
		cache: lru,
		bufPopl: sync.Pool{
			New: func() any {
				return make([]byte, 512)
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
		buf := make([]byte, 512)
		readLen, addr, err := s.listener.ReadFromUDP(buf)
		if err != nil {
			log.Println(err)
			return
		}
		go s.udpHandler(addr, buf[:readLen])
	}
}

func (s *Socket) udpHandler(addr net.Addr, in []byte) {
	parser := dnsmessage.Parser{}

	header, err := parser.Start(in)
	if err != nil {
		log.Println(err)
		return
	}

	question, err := parser.AllQuestions()
	if err != nil {
		log.Println(err)
		return
	}
	//get result from cache
	if answer, ok := s.cache.Get(question[0]); ok {
		buf := s.bufPopl.Get().([]byte)
		msg := dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:       header.ID,
				Response: true,
				RCode:    dnsmessage.RCodeSuccess,
			},
			Questions: question,
			Answers:   answer,
		}
		buf, err = msg.Pack()
		if err != nil {
			log.Println(err)
			return
		}
		_, err = s.listener.WriteTo(buf, addr)
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		//parser2 := dnsmessage.Parser{}
		remoteDns, ok := s.connPoll.Get().(net.Conn)
		defer func() {
			err := remoteDns.Close()
			if err != nil {
				log.Println(err)
			}
		}()
		if !ok {
			log.Println("cant connect to remote dns")
			return
		}
		// redirect the query to remoteDns
		_, err := remoteDns.Write(in)
		if err != nil {
			log.Println(err)
			return
		}

		resp := s.bufPopl.Get().([]byte)

		// read response from remoteDns
		n, err := remoteDns.Read(resp)
		if err != nil {
			log.Println(err)
			return
		}
		// write response to user
		_, err = s.listener.WriteTo(resp[:n], addr)
		if err != nil {
			log.Println(err)
			return
		}

		// start parsing response to add it to the cache
		_, err = parser.Start(resp)
		if err != nil {
			log.Println(err)
			return
		}

		question, err := parser.AllQuestions()
		if err != nil {
			log.Println(err)
			return
		}
		r, err := parser.AllAnswers()
		if err != nil {
			log.Println(err)
			return
		}

		if r != nil {
			s.cache.Add(question[0], r)
		}
	}
}
