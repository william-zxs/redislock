package redislock

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

func getRedisLock() *RedisLock {
	c := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "william", // 没有密码，默认值
		DB:       0,         // 默认DB 0
	})
	return NewRedisLock(c)
}

func workLock(wg *sync.WaitGroup, lock *RedisLock) {
	defer wg.Done()
	uuid, err := lock.Lock("william")
	if err != nil {
		panic(err)
	}
	if uuid != "" {
		fmt.Println("==get lock==", uuid)
		time.Sleep(time.Second * 1)
	} else {
		fmt.Println("==not get lock==", uuid)
	}
	res, err := lock.Unlock("william", uuid)
	if err != nil {
		panic(err)
	}
	if res {
		fmt.Println("==unlock==", uuid)
	} else {
		fmt.Println("==not unlock==", uuid)
	}

}

func workTryLock(wg *sync.WaitGroup, lock *RedisLock) {
	defer wg.Done()
	res, uuid, err := lock.TryLock("william")
	if err != nil {
		panic(err)
	}
	if res {
		fmt.Println("==get lock==", uuid)
		time.Sleep(time.Second * 1)
	} else {
		fmt.Println("==not get lock==", uuid)
	}
	res, err = lock.Unlock("william", uuid)
	if err != nil {
		panic(err)
	}
	if res {
		fmt.Println("==unlock==", uuid)
	} else {
		fmt.Println("==not unlock==", uuid)
	}

}

func TestRedisLock_Lock(t *testing.T) {
	redislock := getRedisLock()

	var a sync.Mutex
	a.TryLock()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go workLock(&wg, redislock)
	}
	wg.Wait()
}

func TestRedisLock_TryLock(t *testing.T) {
	redislock := getRedisLock()

	var a sync.Mutex
	a.TryLock()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go workTryLock(&wg, redislock)
	}
	wg.Wait()
}

func BenchmarkRedisLock_Lock(b *testing.B) {
	redislock := getRedisLock()
	count := 0
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uuid, err := redislock.Lock("william")
			if err != nil {
				panic(err)
			}
			count += 1
			redislock.Unlock("william", uuid)
		}()

	}
	wg.Wait()
	fmt.Println("\n==count==", count, b.N)

}

func BenchmarkMutex(b *testing.B) {
	count := 0
	var wg sync.WaitGroup
	var m sync.Mutex
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Lock()
			count += 1
			m.Unlock()
		}()

	}
	wg.Wait()
	fmt.Println("\n==count==", count, b.N)

}
