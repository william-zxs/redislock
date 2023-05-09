package redislock

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

func getRedisLock() Lockbox {
	return NewRedisLock(redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "william",
		DB:       0,
	}))
}

func workLock(wg *sync.WaitGroup, lock Lock) {
	defer wg.Done()
	err := lock.Lock()
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Millisecond * 100)
	err = lock.Unlock()
	if err != nil {
		panic(err)
	}
}

func workTryLock(wg *sync.WaitGroup, lock Lock) {
	defer wg.Done()
	res, err := lock.TryLock()
	if err != nil {
		panic(err)
	}
	if res {
		fmt.Println("==get lock==!!!!")
		time.Sleep(time.Second * 5)
	} else {
		fmt.Println("==not get lock==")
	}
	err = lock.Unlock()
	if err != nil {
		panic(err)
	}
}

func TestRedisLock_Lock(t *testing.T) {
	redislock := getRedisLock()
	l := redislock.GetLock("william")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go workLock(&wg, l)
	}
	wg.Wait()
}

func TestRedisLock_TryLock(t *testing.T) {
	redislock := getRedisLock()
	l := redislock.GetLock("william")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go workTryLock(&wg, l)
	}
	wg.Wait()
}

func BenchmarkRedisLock_Lock(b *testing.B) {
	redislock := getRedisLock()
	l := redislock.GetLock("william")
	count := 0
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := l.Lock()
			if err != nil {
				panic(err)
			}
			count += 1
			l.Unlock()
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
