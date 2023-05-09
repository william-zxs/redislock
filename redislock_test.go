package redislock

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"testing"
	"time"
)

var lockClient LockClient

func init() {
	lockClient = NewRedisLock(redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "william",
		DB:       0,
	}))
}

func workLock(wg *sync.WaitGroup) {
	l := lockClient.GetLock("william")
	defer wg.Done()
	err := l.Lock()
	if err != nil {
		panic(err)
	}
	fmt.Println("==get lock==!!!!")
	time.Sleep(time.Millisecond * 1000)
	err = l.Unlock()
	if err != nil {
		panic(err)
	}
	fmt.Println("==unlock==")
}

func workTryLock(wg *sync.WaitGroup) {
	lock := lockClient.GetLock("william")
	defer wg.Done()
	res, err := lock.TryLock()
	if err != nil {
		panic(err)
	}
	if res {
		fmt.Println("==get lock==!!!!")
		time.Sleep(time.Second * 20)
	} else {
		fmt.Println("==not get lock==")
	}
	err = lock.Unlock()
	fmt.Println("==unlock==")
	if err != nil {
		panic(err)
	}
}

func TestRedisLock_Lock(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go workLock(&wg)
	}
	wg.Wait()
}

func TestRedisLock_TryLock(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go workTryLock(&wg)
	}
	wg.Wait()
}

func BenchmarkRedisLock_Lock(b *testing.B) {
	count := 0
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := lockClient.GetLock("william", WithTimeout(time.Second*30))
			err := l.Lock()
			if err != nil {
				panic(err)
			}
			count += 1
			l.Unlock()
		}()

	}
	wg.Wait()
	//fmt.Println("\n==count==", count, b.N)

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
	//fmt.Println("\n==count==", count, b.N)

}
