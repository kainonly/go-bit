package kv

import (
	"errors"
	"github.com/bytedance/sonic"
	"github.com/nats-io/nats.go"
	"time"
)

type Service struct {
	*KV
}

// Load 载入配置
func (x *Service) Load() (err error) {
	var b []byte
	var entry nats.KeyValueEntry
	if entry, err = x.KeyValue.Get("values"); err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			if b, err = sonic.Marshal(x.DynamicValues); err != nil {
				return
			}
			if _, err = x.KeyValue.Put("values", b); err != nil {
				return
			}
		} else {
			return
		}
	}

	if b == nil {
		b = entry.Value()
	}

	if err = sonic.Unmarshal(b, &x.DynamicValues); err != nil {
		return
	}

	return
}

// Sync 同步节点动态配置
func (x *Service) Sync() (err error) {
	if err = x.Load(); err != nil {
		return
	}

	var watch nats.KeyWatcher
	if watch, err = x.KeyValue.Watch("values"); err != nil {
		return
	}

	current := time.Now()
	for entry := range watch.Updates() {
		if entry == nil || entry.Created().Unix() < current.Unix() {
			continue
		}
		// 同步动态配置
		if err = sonic.Unmarshal(entry.Value(), &x.DynamicValues); err != nil {
			return
		}
	}

	return
}

// Set 设置动态配置
func (x *Service) Set(update map[string]interface{}) (err error) {
	var entry nats.KeyValueEntry
	if entry, err = x.KeyValue.Get("values"); err != nil {
		return
	}
	var values map[string]interface{}
	if err = sonic.Unmarshal(entry.Value(), &values); err != nil {
		return
	}
	for k, v := range update {
		values[k] = v
	}
	return x.Update(values)
}

var SECRET = map[string]bool{
	"tencent_secret_key":        true,
	"feishu_app_secret":         true,
	"feishu_encrypt_key":        true,
	"feishu_verification_token": true,
	"email_password":            true,
	"openapi_secret":            true,
}

// Get 获取动态配置
func (x *Service) Get(keys ...string) (values map[string]interface{}, err error) {
	var entry nats.KeyValueEntry
	if entry, err = x.KeyValue.Get("values"); err != nil {
		return
	}
	if err = sonic.Unmarshal(entry.Value(), &values); err != nil {
		return
	}
	sets := make(map[string]bool)
	for _, key := range keys {
		sets[key] = true
	}
	all := len(sets) == 0
	for k, v := range values {
		if !all && !sets[k] {
			continue
		}
		if SECRET[k] {
			// 密文
			if v != nil || v != "" {
				values[k] = "*"
			} else {
				values[k] = "-"
			}
		} else {
			values[k] = v
		}
	}
	return
}

// Remove 移除动态配置
func (x *Service) Remove(key string) (err error) {
	var entry nats.KeyValueEntry
	if entry, err = x.KeyValue.Get("values"); err != nil {
		return
	}
	var values map[string]interface{}
	if err = sonic.Unmarshal(entry.Value(), &values); err != nil {
		return
	}
	delete(values, key)
	return x.Update(values)
}

// Update 更新配置
func (x *Service) Update(values map[string]interface{}) (err error) {
	var b []byte
	if b, err = sonic.Marshal(values); err != nil {
		return
	}
	if _, err = x.KeyValue.Put("values", b); err != nil {
		return
	}
	return
}
