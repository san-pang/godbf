package godbf

import (
	"io"
	"os"
	"syscall"
)

// This used to call syscall.Flock() but that call fails with EBADF on NFS.
// An alternative is lockf() which works on NFS but that call lets a process
// lock the same file twice. Instead, use Linux's non-standard open file
// descriptor locks which will block if the process already holds the file lock.
//
// constants from /usr/include/bits/fcntl-linux.h
const (
	F_OFD_GETLK  = 37
	F_OFD_SETLK  = 37
	F_OFD_SETLKW = 38
)

var (
	wrlck = syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	unlck = syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}

	linuxTryLockFile = flockTryLockFile
	linuxLockFile    = flockLockFile
	linuxUnlockFile  = flockUnlockFile
)

func init() {
	// use open file descriptor locks if the system supports it
	getlk := syscall.Flock_t{Type: syscall.F_RDLCK}
	if err := syscall.FcntlFlock(0, F_OFD_GETLK, &getlk); err == nil {
		linuxTryLockFile = ofdTryLockFile
		linuxLockFile = ofdLockFile
		linuxUnlockFile = ofdUnlockFile
	}
}

type lock struct {
	file *os.File
}

// New creates a new lock
func newlock(file *os.File) tryLockerSafe {
	l := &lock{
		file: file,
	}
	return l
}

// TryLock acquires exclusivity on the lock without blocking
func (l *lock) tryLock() (bool, error) {
	return linuxTryLockFile(l.file.Fd())
}

// Lock acquires exclusivity on the lock without blocking
func (l *lock) lock() error {
	return linuxLockFile(l.file.Fd())
}

// Unlock unlocks the lock
func (l *lock) unlock() error {
	return linuxUnlockFile(l.file.Fd())
}

func ofdTryLockFile(fd int) (bool, error) {
	flock := wrlck
	if err := syscall.FcntlFlock(uintptr(fd), F_OFD_SETLK, &flock); err != nil {
		if err == syscall.EWOULDBLOCK {
			return false, errLocked
		}
		return false, err
	}
	return true, nil
}

func ofdLockFile(fd int) error {
	flock := wrlck
	return syscall.FcntlFlock(uintptr(fd), F_OFD_SETLKW, &flock)
}

func ofdUnlockFile(fd int) error {
	flock := unlck
	return syscall.FcntlFlock(uintptr(fd), F_OFD_SETLKW, &flock)
}

// Check the interfaces are satisfied
var (
	_ TryLockerSafe = &lock{}
)
