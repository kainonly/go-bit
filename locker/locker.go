package locker

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type Locker struct {
	Namespace string
	Redis     *redis.Client
}

func New(options ...Option) *Locker {
	x := new(Locker)
	for _, v := range options {
		v(x)
	}
	return x
}

type Option func(x *Locker)

func SetNamespace(v string) Option {
	return func(x *Locker) {
		x.Namespace = v
	}
}

func SetRedis(v *redis.Client) Option {
	return func(x *Locker) {
		x.Redis = v
	}
}

func (x *Locker) Key(name string) string {
	return fmt.Sprintf(`%s:locker:%s`, x.Namespace, name)
}

func (x *Locker) Update(ctx context.Context, name string, ttl time.Duration) (err error) {
	key := x.Key(name)
	var exists int64
	if exists, err = x.Redis.
		Exists(ctx, key).
		Result(); err != nil {
		return
	}

	if exists == 0 {
		if err = x.Redis.
			Set(ctx, key, 1, ttl).
			Err(); err != nil {
			return
		}
	} else {
		if err = x.Redis.
			Incr(ctx, key).
			Err(); err != nil {
			return
		}
	}
	return
}

func (x *Locker) Verify(ctx context.Context, name string, n int64) (result bool, err error) {
	key := x.Key(name)
	var exists int64
	if exists, err = x.Redis.Exists(ctx, key).Result(); err != nil {
		return
	}
	if exists == 0 {
		return
	}

	var count int64
	if count, err = x.Redis.Get(ctx, key).Int64(); err != nil {
		return
	}

	return count >= n, nil
}

func (x *Locker) Delete(ctx context.Context, name string) error {
	return x.Redis.Del(ctx, x.Key(name)).Err()
}