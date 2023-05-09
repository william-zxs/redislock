# redislock

redislock is a distributed lock implementation based on redis

## usage
```go

// init redislock
redislock := NewRedisLock(redis.NewClient(&redis.Options{
                Addr:     "localhost:6379",
                Password: "",
                DB:       0,
            }))
// get lock
uuid,err := redislock.Lock("helloworld")
//do something
// unlock
redislock.Unlock("helloworld",uuid)

```



每一个goroutine应该重新初始化

## todo
2. 效率更好的自旋实现（公平锁、非公平锁）
3. 重入锁


