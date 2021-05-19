# autolock

**Package autolock implements a small wrapper over `github.com/bsm/redis-lock` to automatically refresh locks.**

## Example

```go
// create client
client := redis.NewClient(&redis.Options{
    Addr: "0.0.0.0:6379",
})
defer client.Close()

// acquire lock
lock, err := autolock.Acquire(client, "lock-key", &Options{
    LockTimeout:     time.Second,
    RefreshInterval: 10 * time.Millisecond,
})
if err != nil {
    panic(err)
}

// print status
fmt.Println(lock.Status())

// do some work
time.Sleep(100 * time.Millisecond)

// print status
fmt.Println(lock.Status())

// release lock
err = lock.Release()
if err != nil {
    panic(err)
}

// print status
fmt.Println(lock.Status())

// Output:
// true <nil>
// true <nil>
// false <nil>
```
