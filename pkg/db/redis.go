package db

import "github.com/WuKongIM/WuKongIMBusinessExtra/pkg/redis"

func NewRedis(addr string, password string) *redis.Conn {
	return redis.New(addr, password)
}
