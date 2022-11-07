// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package filemutex

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	mkdirPerm = 0750
)

// FileMutex is similar to sync.RWMutex, but also synchronizes across processes.
// This implementation is based on flock syscall.
type FileMutex struct {
	fd int
}

func New(filename string) (*FileMutex, error) {
	return new(filename, mkdirPerm)
}

func NewWithUnixFilePerm(filename string, perm uint32) (*FileMutex, error) {
	return new(filename, perm)
}

func new(filename string, perm uint32) (*FileMutex, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.FileMode(perm)); err != nil {
		return nil, err
	}
	fd, err := unix.Open(filename, unix.O_CREAT|unix.O_RDONLY, perm)
	if err != nil {
		return nil, err
	}
	return &FileMutex{fd: fd}, nil
}

func (m *FileMutex) Lock() error {
	return unix.Flock(m.fd, unix.LOCK_EX)
}

func (m *FileMutex) TryLock() error {
	if err := unix.Flock(m.fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errno, ok := err.(unix.Errno); ok {
			if errno == unix.EWOULDBLOCK {
				return AlreadyLocked
			}
		}
		return err
	}
	return nil
}

func (m *FileMutex) Unlock() error {
	return unix.Flock(m.fd, unix.LOCK_UN)
}

func (m *FileMutex) RLock() error {
	return unix.Flock(m.fd, unix.LOCK_SH)
}

func (m *FileMutex) RUnlock() error {
	return unix.Flock(m.fd, unix.LOCK_UN)
}

// Close unlocks the lock and closes the underlying file descriptor.
func (m *FileMutex) Close() error {
	if err := unix.Flock(m.fd, unix.LOCK_UN); err != nil {
		return err
	}
	return unix.Close(m.fd)
}
