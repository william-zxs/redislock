# redislock

redislock is a Distributed lock implementation based on Redis

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
redislock.Unlock(uuid)

```



## todo
1. watchdog
2. 效率更好的自旋实现（公平锁、非公平锁）
3. 重入锁


