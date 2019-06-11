package autolock

import (
	"testing"
	"time"

	"github.com/go-redis/redis"

	"github.com/stretchr/testify/assert"
)

var testClient *redis.Client

func init() {
	testClient = redis.NewClient(&redis.Options{
		Addr: "0.0.0.0:6379",
	})
}

func TestLocking(t *testing.T) {
	opts := &Options{
		LockTimeout:     time.Second,
		RefreshInterval: 10 * time.Millisecond,
	}

	lck1, err := Acquire(testClient, "autolock.test1", opts)
	assert.NoError(t, err)
	assert.NotNil(t, lck1)

	lck2, err := Acquire(testClient, "autolock.test1", opts)
	assert.NoError(t, err)
	assert.Nil(t, lck2)

	list, err := testClient.Keys("autolock.*").Result()
	assert.NoError(t, err)
	assert.Equal(t, []string{"autolock.test1"}, list)

	err = lck1.Release()
	assert.NoError(t, err)

	err = lck1.Release()
	assert.NoError(t, err)

	lck2, err = Acquire(testClient, "autolock.test1", opts)
	assert.NoError(t, err)
	assert.NotNil(t, lck2)

	err = lck2.Release()
	assert.NoError(t, err)

	list, err = testClient.Keys("autolock.*").Result()
	assert.NoError(t, err)
	assert.Empty(t, list)
}

func TestRefreshing(t *testing.T) {
	opts := &Options{
		LockTimeout:     time.Second,
		RefreshInterval: 10 * time.Millisecond,
	}

	lck, err := Acquire(testClient, "autolock.test2", opts)
	assert.NoError(t, err)
	assert.NotNil(t, lck)

	time.Sleep(25 * time.Millisecond)

	ok, err := lck.Status()
	assert.NoError(t, err)
	assert.True(t, ok)

	err = lck.Release()
	assert.NoError(t, err)

	ok, err = lck.Status()
	assert.NoError(t, err)
	assert.False(t, ok)

	list, err := testClient.Keys("autolock.*").Result()
	assert.NoError(t, err)
	assert.Empty(t, list)
}

func TestTimeout(t *testing.T) {
	lck1, err := Acquire(testClient, "autolock.test3", &Options{
		LockTimeout:     time.Millisecond,
		RefreshInterval: 10 * time.Millisecond,
	})
	assert.NoError(t, err)
	assert.NotNil(t, lck1)

	time.Sleep(5 * time.Millisecond)

	lck2, err := Acquire(testClient, "autolock.test3", &Options{
		LockTimeout:     time.Second,
		RefreshInterval: 10 * time.Millisecond,
	})
	assert.NoError(t, err)
	assert.NotNil(t, lck2)

	<-lck1.Abandoned()

	ok, err := lck1.Status()
	assert.Error(t, err)
	assert.False(t, ok)

	err = lck2.Release()
	assert.NoError(t, err)

	list, err := testClient.Keys("autolock.*").Result()
	assert.NoError(t, err)
	assert.Empty(t, list)
}

func Benchmark(b *testing.B) {
	opts := &Options{
		LockTimeout:     time.Second,
		RefreshInterval: 10 * time.Millisecond,
	}

	for i := 0; i < b.N; i++ {
		lck1, err := Acquire(testClient, "autolock.bench1", opts)
		if err != nil {
			panic(err)
		}

		err = lck1.Release()
		if err != nil {
			panic(err)
		}
	}
}
