package godbf

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// see https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx
	flagLockExclusive       = 2
	flagLockFailImmediately = 1

	// see https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
	errLockViolation syscall.Errno = 0x21
)

type lock struct {
	file *os.File
}

// New creates a new lock
func newLock(file *os.File) tryLockerSafe {
	l := &lock{
		file: file,
	}
	return l
}

// TryLock acquires exclusivity on the lock without blocking
func (l *lock) tryLock() (bool, error) {
	return lockFile(syscall.Handle(l.file.Fd()), flagLockFailImmediately)
}

// Lock acquires exclusivity on the lock without blocking
func (l *lock) lock() error {
	_, err := lockFile(syscall.Handle(l.file.Fd()), 0)
	return err
}

// Unlock unlocks the lock
func (l *lock) unlock() error {
	err := unlockFileEx(syscall.Handle(l.file.Fd()), 0, 1, 0, &syscall.Overlapped{})
	return err
}

func lockFile(fd syscall.Handle, flags uint32) (bool, error) {
	var flag uint32 = flagLockExclusive
	flag |= flags
	if fd == syscall.InvalidHandle {
		return true, nil
	}
	err := lockFileEx(fd, flag, 0, 1, 0, &syscall.Overlapped{})
	if err == nil {
		return true, nil
	} else if err.Error() == errLocked.Error() {
		return false, errLocked
	} else if err != errLockViolation {
		return false, err
	}
	return true, nil
}

func lockFileEx(h syscall.Handle, flags, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) (err error) {
	r1, _, e1 := syscall.Syscall6(procLockFileEx.Addr(), 6, uintptr(h), uintptr(flags), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)))
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func unlockFileEx(h syscall.Handle, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) (err error) {
	r1, _, e1 := syscall.Syscall6(procUnlockFileEx.Addr(), 5, uintptr(h), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// Check the interfaces are satisfied
var (
	_ tryLockerSafe = &lock{}
)