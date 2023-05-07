package redislock

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisLock struct {
	c *redis.Client
}

func (r *RedisLock) Lock(key string) (string, error) {
	ctx := context.Background()
	uuid := uuid.New().String()
	t := time.NewTimer(time.Second * 100)
	defer t.Stop()
	//自旋，等待获得锁
	for {
		//这样做不好，太耗费资源了，如果能watch key的变化就好了。
		res, err := r.c.SetNX(ctx, key, uuid, time.Second*60).Result()
		if err != nil {
			return "", err
		}
		if res {
			return uuid, nil
		}

		select {
		case <-t.C:
			return "", errors.New("redis lock wait time out!")
		default:
			// sleep 10ms
			time.Sleep(time.Millisecond * 1)
		}

	}

}

// ctx的使用
func (r *RedisLock) TryLock(key string) (bool, string, error) {
	ctx := context.Background()
	uuid := uuid.New().String()
	res, err := r.c.SetNX(ctx, key, uuid, time.Second*60).Result()
	if err != nil || !res {
		return false, "", err
	}

	return res, uuid, nil
}

func (r *RedisLock) Unlock(key string, uuid string) (bool, error) {
	// 判断当前锁的状态
	// 1.如果锁存在且是当前线程的锁，删除key，返回true，需要用到lua原子操作。
	// 2.如果锁不存在或者不是当前线程拥有的锁，返回false
	//fmt.Println("==uuid==", uuid)
	if uuid == "" {
		return false, nil
	}
	ctx := context.Background()
	var unlock = redis.NewScript(`
		local key = KEYS[1]
		local currentId = ARGV[1]
		
		local value = redis.call("GET", key)
		if not value then
		  value = 0
		end
		if value == currentId then 
			redis.call("DEL", key)
		end
		return value
		`)
	res, err := unlock.Run(ctx, r.c, []string{key}, uuid).Result()
	res = res.(string)
	if res == "0" || res != uuid {
		return false, err
	}

	return true, err

}

func NewRedisLock(c *redis.Client) *RedisLock {
	return &RedisLock{
		c: c,
	}
}
