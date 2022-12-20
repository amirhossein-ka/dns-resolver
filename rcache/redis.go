package rcache

import (
	"context"
	"dns-resolver/args"
	"github.com/go-redis/redis/v9"
)

type Rdb struct {
	c *redis.Client
}

func New(r args.Redis) (*Rdb, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     r.Addr,
		Password: r.Password,
		DB:       r.DB,
	})

	if s := c.Ping(context.Background()); s.Err() != nil {
		return nil, s.Err()
	}
	return &Rdb{c: c}, nil
}
