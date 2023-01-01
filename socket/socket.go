package socket

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
		args     args.SocketArgs
		mu       sync.Mutex
		cache    *cache.LRU[dnsmessage.Question, []dnsmessage.Resource]
		bufPoll  sync.Pool
		connPoll sync.Pool
		listener *net.UDPConn
		queue    Queue
	}
	Queue chan QueueRequest

	QueueRequest struct {
		Data   []byte
		Addr   net.Addr
		Length int
	}
)

func NewSocket(args args.SocketArgs) (*Socket, error) {
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
		args:  args,
		mu:    sync.Mutex{},
		cache: lru,
		bufPoll: sync.Pool{
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
		queue:    make(Queue, args.Workers*4),
	}, nil
}

// ListenAndServe is a non blocking call,
func (s *Socket) ListenAndServe() {
	for i := 0; i < s.args.Workers; i++ {
		go s.dequeuer()
		go s.reader()
	}
}

func (s *Socket) reader() {
	for {
		buf := s.bufPoll.Get().([]byte)
		n, addr, err := s.listener.ReadFromUDP(buf[0:])
		if err != nil {
			log.Println(err)
			continue
		}

		s.queue <- QueueRequest{
			Data:   buf,
			Addr:   addr,
			Length: n,
		}
	}
}

// suggest a better name for this
func (s *Socket) dequeuer() {
	for req := range s.queue {
		s.udpHandler(req.Addr, req.Data[:req.Length])
        	s.bufPoll.Put(req.Data)
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
		responseByte := s.bufPoll.Get().([]byte)
		msg := dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:       header.ID,
				Response: true,
				RCode:    dnsmessage.RCodeSuccess,
			},
			Questions: question,
			Answers:   answer,
		}
		responseByte, err = msg.Pack()
		if err != nil {
			log.Println(err)
			return
		}
		_, err = s.listener.WriteTo(responseByte, addr)
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

		resp := s.bufPoll.Get().([]byte)

		// read response from remoteDns
		n, err := remoteDns.Read(resp[0:])
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
