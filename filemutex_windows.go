// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filemutex

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock = 2
)

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

// FileMutex is similar to sync.RWMutex, but also synchronizes across processes.
// This implementation is based on flock syscall.
type FileMutex struct {
	mu sync.RWMutex
	fd syscall.Handle
}

func New(filename string) (*FileMutex, error) {
	fd, err := syscall.CreateFile(&(syscall.StringToUTF16(filename)[0]), syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return &FileMutex{fd: fd}, nil
}

func (m *FileMutex) Lock() {
	m.mu.Lock()
	var ol syscall.Overlapped
	if err := lockFileEx(m.fd, lockfileExclusiveLock, 0, 1, 0, &ol); err != nil {
		panic(err)
	}
}

func (m *FileMutex) Unlock() {
	var ol syscall.Overlapped
	if err := unlockFileEx(m.fd, 0, 1, 0, &ol); err != nil {
		panic(err)
	}
	m.mu.Unlock()
}

func (m *FileMutex) RLock() {
	m.mu.RLock()
	var ol syscall.Overlapped
	if err := lockFileEx(m.fd, 0, 0, 1, 0, &ol); err != nil {
		panic(err)
	}
}

func (m *FileMutex) RUnlock() {
	var ol syscall.Overlapped
	if err := unlockFileEx(m.fd, 0, 1, 0, &ol); err != nil {
		panic(err)
	}
	m.mu.RUnlock()
}
