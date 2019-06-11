// Package autolock implements a small wrapper over github.com/bsm/redis-lock
// to automatically refresh locks.
package autolock

import (
	"errors"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"gopkg.in/tomb.v2"
)

// ErrLostLock is returned if the lock has been lost to another process.
var ErrLostLock = errors.New("lost lock")

// Lock is returned by Acquire and represents an active lock.
type Lock struct {
	locker   *lock.Locker
	interval time.Duration
	tomb     tomb.Tomb
}

func createLock(locker *lock.Locker, interval time.Duration) *Lock {
	// prepare lock
	lck := &Lock{
		locker:   locker,
		interval: interval,
	}

	// run keeper
	lck.tomb.Go(lck.keeper)

	return lck
}

func (l *Lock) keeper() error {
	for {
		select {
		case <-time.After(l.interval):
			// refresh lock
			ok, err := l.locker.Lock()
			if err != nil {
				return err
			} else if !ok {
				return ErrLostLock
			}
		case <-l.tomb.Dying():
			// release lock
			err := l.locker.Unlock()
			if err != nil {
				return err
			}

			return tomb.ErrDying
		}
	}
}

// Abandoned returns a channel that is closed when the lock has been abandoned.
// The closing of this channel does not necessarily mean that the lock has been
// successfully released.
func (l *Lock) Abandoned() <-chan struct{} {
	return l.tomb.Dying()
}

// Status reports the state of the lock.
func (l *Lock) Status() (bool, error) {
	// check error
	err := l.tomb.Err()
	if err == tomb.ErrStillAlive {
		return true, nil
	}

	return false, err
}

// Release will release the lock.
func (l *Lock) Release() error {
	l.tomb.Kill(nil)
	return l.tomb.Wait()
}

// Options is used to configure the lock acquisition, refresh and release.
type Options struct {
	// The time after a lock will release itself.
	//
	// Default: 5s
	LockTimeout time.Duration

	// The amount of initial retries to acquire the lock.
	//
	// Default: 0
	RetryCount int

	// The delay between individual attempts to acquire the lock.
	//
	// Default: 100ms
	RetryDelay time.Duration

	// The interval of the refresh cycle. Should be considerably smaller than
	// the LockTimeout to ensure the lock is refreshed.
	//
	// Default: LockTimeout / 2 (2.5s)
	RefreshInterval time.Duration
}

// Acquire will attempt to the lock represented by the specified key. It will
// retry the acquisition using the configured delay. The returned lock is
// automatically refreshed until it is released. If the lock attempt failed nil
// is returned.
func Acquire(client *redis.Client, key string, options *Options) (*Lock, error) {
	// ensure default
	if options == nil {
		options = &Options{}
	}

	// set default lock timeout
	if options.LockTimeout <= 0 {
		options.LockTimeout = 5 * time.Second
	}

	// set default retry count
	if options.RetryCount < 0 {
		options.RetryCount = 0
	}

	// set default retry delay
	if options.RetryDelay <= 0 {
		options.RetryDelay = 100 * time.Millisecond
	}

	// set default refresh interval
	if options.RefreshInterval <= 0 {
		options.RefreshInterval = options.LockTimeout / 2 // 2.5s
	}

	// attempt to obtain lock
	locker, err := lock.Obtain(client, key, &lock.Options{
		RetryCount:  options.RetryCount,
		RetryDelay:  options.RetryDelay,
		LockTimeout: options.LockTimeout,
	})
	if err == lock.ErrLockNotObtained {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// create lock
	lck := createLock(locker, options.RefreshInterval)

	return lck, nil
}
