package redislock

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisLock struct {
	*redis.Client
}

type lock struct {
	c       *redisLock
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

type LockClient interface {
	GetLock(key string, options ...Option) Lock
}

type Option func(*lock)

func WithTimeout(timeout time.Duration) Option {
	return func(r *lock) {
		r.timeout = timeout
	}
}
func WithTTL(ttl time.Duration) Option {
	return func(r *lock) {
		r.ttl = ttl
	}
}

// option 模式
func NewRedisLock(c *redis.Client) LockClient {
	//默认参数
	lock := &redisLock{
		c,
	}
	return lock
}

func (r *redisLock) GetLock(key string, options ...Option) Lock {
	//todo 是否可以使用对象池优化
	ctx, cancel := context.WithCancel(context.Background())
	l := &lock{
		c:       r,
		timeout: time.Second * 10,
		ttl:     time.Second * 30,
		key:     key,
		uuid:    uuid.New().String(),
		ctx:     ctx,
		cancel:  cancel,
	}
	//自定义参数
	for _, option := range options {
		option(l)
	}
	return l
}

// todo ctx如何使用
func (r *lock) TryLock() (bool, error) {
	res, err := r.c.SetNX(r.ctx, r.key, r.uuid, r.ttl).Result()
	if err != nil || !res {
		return false, err
	}
	go r.startWatchdog()
	return res, nil
}

func (l *lock) Lock() error {
	//等待超时时间
	t := time.NewTimer(l.timeout)
	defer t.Stop()
	//自旋，等待获得锁
	for {
		//一直try耗费资源，如果能watch key的变化就好了。
		res, err := l.TryLock()
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

func (l *lock) Unlock() error {
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
	res, err := unlock.Run(ctx, l.c, []string{l.key}, l.uuid).Result()
	if res.(int64) == 1 {
		//停掉watch dog
		l.cancel()
	}
	return err

}

// 开一个goroutine 去给key刷新ttl
func (l *lock) startWatchdog() {
	ticker := time.NewTicker(l.ttl / 3)
	for {
		select {
		case <-ticker.C:
			//刷新ttl
			err := l.c.Expire(l.ctx, l.key, l.ttl).Err()
			if err != nil {
				return
			}
		case <-l.ctx.Done():
			ticker.Stop()
			return
		}
	}
}
