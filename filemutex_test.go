package filemutex

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockUnlock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m.Lock()
	m.Unlock()
}

func TestRLockUnlock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m.RLock()
	m.RUnlock()
}

func TestClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m.Lock()
	m.Close()
}

func TestSequentialLock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m.Lock()

	state := "waiting"
	ch := make(chan struct{})
	go func() {
		m.Lock()
		state = "acquired"
		ch <- struct{}{}

		<-ch
		m.Unlock()
		state = "released"
		ch <- struct{}{}
	}()

	assert.Equal(t, "waiting", state)
	m.Unlock()

	<-ch
	assert.Equal(t, "acquired", state)
	ch <- struct{}{}

	<-ch
	assert.Equal(t, "released", state)
}

func TestSimultaneousLock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m2, err := New(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err)

	m.Lock()

	state := "waiting"
	ch := make(chan struct{})
	go func() {
		go func() {
			<-time.After(50 * time.Millisecond)
			state = "waiting on lock" // for m to unlock before m2 can lock
			ch <- struct{}{}
		}()
		m2.Lock()
		state = "acquired"
		ch <- struct{}{}

		<-ch
		m2.Unlock()
		state = "closed"
		ch <- struct{}{}
	}()

	assert.Equal(t, "waiting", state)

	<-ch
	assert.Equal(t, "waiting on lock", state)

	m.Unlock()

	<-ch
	assert.Equal(t, "acquired", state)
	ch <- struct{}{}

	<-ch
	assert.Equal(t, "closed", state)

	_, err = os.Stat(path)
	assert.Nil(t, err)
}

func TestSimultaneousLockWithClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	m2, err := New(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err)

	m.Lock()

	state := "waiting"
	ch := make(chan struct{})
	go func() {
		go func() {
			<-time.After(50 * time.Millisecond)
			state = "waiting on lock" // for m to unlock before m2 can lock
			ch <- struct{}{}
		}()
		m2.Lock()
		state = "acquired"
		ch <- struct{}{}

		<-ch
		m2.Close()
		state = "closed"
		ch <- struct{}{}
	}()

	assert.Equal(t, "waiting", state)

	<-ch
	assert.Equal(t, "waiting on lock", state)

	m.Close()

	<-ch
	assert.Equal(t, "acquired", state)
	ch <- struct{}{}

	<-ch
	assert.Equal(t, "closed", state)

	_, err = os.Stat(path)
	assert.NotNil(t, err)
}
