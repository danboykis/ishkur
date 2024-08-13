package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/danboykis/ishkur/config"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net"
	"time"
)

type Db interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	Close(ctx context.Context) error
}

type RedisDb struct {
	rdb *redis.Client
}

var (
	NotFoundError = errors.New("key not found")
	InternalError = errors.New("internal error")
)

func swapError(err error) error {
	if err != nil {
		return InternalError
	}
	return nil
}

func (rdb *RedisDb) Get(ctx context.Context, key string) (string, error) {
	r, err := rdb.rdb.Get(ctx, key).Result()
	if err != nil {
		switch {
		case errors.Is(err, redis.Nil):
			return "", NotFoundError
		default:
			return "", InternalError
		}
	}

	return r, nil
}

func (rdb *RedisDb) Set(ctx context.Context, key string, value string) error {
	return swapError(rdb.rdb.Set(ctx, key, value, 0).Err())
}

func (rdb *RedisDb) Close(_ context.Context) error {
	return swapError(rdb.rdb.Close())
}

func NewRedis(c config.Redis) (*RedisDb, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(c.Host, fmt.Sprintf("%d", c.Port)),
		Password: c.Password,
		DB:       0, // use default DB
	})

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	err := swapError(rdb.Ping(ctx).Err())
	defer cancelFn()
	slog.Info("connected to redis", "host", c.Host, "port", c.Port)
	return &RedisDb{rdb: rdb}, err
}
