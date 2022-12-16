package captcha

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/errors"
	"github.com/go-redis/redis/v8"
	"time"
)

type Captcha struct {
	Namespace string
	Redis     *redis.Client
}

func New(options ...Option) *Captcha {
	x := new(Captcha)
	for _, v := range options {
		v(x)
	}
	return x
}

type Option func(x *Captcha)

func SetNamespace(v string) Option {
	return func(x *Captcha) {
		x.Namespace = v
	}
}

func SetRedis(v *redis.Client) Option {
	return func(x *Captcha) {
		x.Redis = v
	}
}

func (x *Captcha) Key(name string) string {
	return fmt.Sprintf(`%s:captcha:%s`, x.Namespace, name)
}

func (x *Captcha) Create(ctx context.Context, name string, code string, ttl time.Duration) error {
	return x.Redis.
		Set(ctx, x.Key(name), code, ttl).
		Err()
}

func (x *Captcha) Exists(ctx context.Context, name string) (_ bool, err error) {
	var exists int64
	if exists, err = x.Redis.Exists(ctx, x.Key(name)).Result(); err != nil {
		return
	}
	return exists != 0, nil
}

var (
	ErrCaptchaNotExists    = errors.NewPublic("the captcha does not exists")
	ErrCaptchaInconsistent = errors.NewPublic("tha captcha is invalid")
)

func (x *Captcha) Verify(ctx context.Context, name string, code string) (err error) {
	var exists bool
	if exists, err = x.Exists(ctx, name); err != nil {
		return
	}
	if !exists {
		return ErrCaptchaNotExists
	}

	var value string
	if value, err = x.Redis.Get(ctx, x.Key(name)).Result(); err != nil {
		return
	}
	if value != code {
		return ErrCaptchaInconsistent
	}

	return
}

func (x *Captcha) Delete(ctx context.Context, name string) error {
	return x.Redis.Del(ctx, x.Key(name)).Err()
}
