package godbf

import (
	"sync"
)

// TryLocker is a sync.Locker augmented with TryLock.
type tryLocker interface {
	sync.Locker
	// TryLock attempts to grab the lock, but does not hang if the lock is
	// actively held by another process.  Instead, it returns false.
	tryLock() bool
}

// TryLockerSafe is like TryLocker, but the methods can return an error and
// never panic.
type tryLockerSafe interface {
	// TryLock attempts to grab the lock, but does not hang if the lock is
	// actively held by another process.  Instead, it returns false.
	tryLock() (bool, error)
	// Lock blocks until it's able to grab the lock.
	lock() error
	// Unlock releases the lock.  Should only be called when the lock is
	// held.
	unlock() error
}

type mustLock struct {
	l tryLockerSafe
}

// Lock implements sync.Locker.Lock.
func (m *mustLock) Lock() {
	if err := m.l.lock(); err != nil {
		panic(err)
	}
}

// TryLock implements TryLocker.TryLock.
func (m *mustLock) tryLock() bool {
	got, err := m.l.tryLock()
	if err != nil {
		panic(err)
	}
	return got
}

// Unlock implements sync.Locker.Unlock.
func (m *mustLock) Unlock() {
	if err := m.l.unlock(); err != nil {
		panic(err)
	}
}

// Check the interfaces are satisfied
var (
	_ tryLocker = &mustLock{}
)