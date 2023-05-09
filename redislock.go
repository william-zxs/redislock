package redislock

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisLock struct {
	c       *redis.Client
	timeout time.Duration
	ttl     time.Duration
	uuid    string
	key     string
	ctx     context.Context
	cancel  context.CancelFunc
}

type Lock interface {
	Lock() error
	TryLock() (bool, error)
	Unlock() error
}

type Lockbox interface {
	GetLock(key string) Lock
}

type Option func(*redisLock)

func WithTimeout(timeout time.Duration) Option {
	return func(r *redisLock) {
		r.timeout = timeout
	}
}
func WithTTL(ttl time.Duration) Option {
	return func(r *redisLock) {
		r.ttl = ttl
	}
}

// option 模式
func NewRedisLock(c *redis.Client, options ...Option) Lockbox {
	//默认参数
	lock := &redisLock{
		c:       c,
		timeout: time.Second * 10,
		ttl:     time.Second * 30,
		ctx:     context.Background(),
	}
	//自定义参数
	for _, option := range options {
		option(lock)
	}
	return lock
}

func (r *redisLock) GetLock(key string) Lock {
	r.key = key
	r.uuid = uuid.New().String()
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r
}

// todo ctx的使用
func (r *redisLock) TryLock() (bool, error) {
	res, err := r.c.SetNX(r.ctx, r.key, r.uuid, r.ttl).Result()
	if err != nil || !res {
		return false, err
	}
	go r.startWatchdog()
	return res, nil
}

func (r *redisLock) Lock() error {
	//等待超时时间
	t := time.NewTimer(r.timeout)
	defer t.Stop()
	//自旋，等待获得锁
	for {
		//一直try耗费资源，如果能watch key的变化就好了。
		res, err := r.TryLock()
		if err != nil {
			return err
		}
		if res {
			return nil
		}
		select {
		case <-t.C:
			return errors.New("redis lock wait time out!")
		default:
		}

	}

}

func (r *redisLock) Unlock() error {
	// 判断当前锁的状态
	// 1.如果锁存在且是当前线程的锁，删除key，需要用到lua原子操作。
	// 2.如果锁不存在或者不是当前线程拥有的锁，返回
	ctx := context.Background()
	var unlock = redis.NewScript(`
		if redis.call("get",KEYS[1]) == ARGV[1] then
    		return redis.call("del",KEYS[1])
		else
    		return 0
		end
		`)
	err := unlock.Run(ctx, r.c, []string{r.key}, r.uuid).Err()
	//停掉watch dog
	r.cancel()
	return err

}

// 开一个goroutine 去给key刷新ttl
func (r *redisLock) startWatchdog() {
	ticker := time.NewTicker(r.ttl / 3)
	for {
		select {
		case <-ticker.C:
			//刷新ttl
			err := r.c.Expire(r.ctx, r.key, r.ttl).Err()
			if err != nil {
				break
			}
		case <-r.ctx.Done():
			ticker.Stop()
			fmt.Println("watchdog stopped...")
			return
		}
	}
}
