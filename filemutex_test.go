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

	err = m.Lock()
	require.NoError(t, err)
	err = m.Unlock()
	require.NoError(t, err)
}

func TestRLockUnlock(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	err = m.RLock()
	require.NoError(t, err)
	err = m.RUnlock()
	require.NoError(t, err)
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

	err = m.Lock()
	require.NoError(t, err)

	state := "waiting"
	ch := make(chan struct{})
	go func() {
		err = m.Lock()
		require.NoError(t, err)
		state = "acquired"
		ch <- struct{}{}

		<-ch
		err = m.Unlock()
		require.NoError(t, err)
		state = "released"
		ch <- struct{}{}
	}()

	assert.Equal(t, "waiting", state)
	err = m.Unlock()
	require.NoError(t, err)

	<-ch
	assert.Equal(t, "acquired", state)
	ch <- struct{}{}

	<-ch
	assert.Equal(t, "released", state)
}

func TestLockErrorsAreRecoverable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	// muck with the internal state in order to cause an error
	oldfd := m.fd
	m.fd = 99999

	// trigger an error
	err = m.Lock()
	require.Error(t, err)

	// restore a sane internal state
	m.fd = oldfd

	// this would hang if we hadn't Unlock()ed in the Lock error branch
	err = m.Lock()
	require.NoError(t, err)

	// clean up
	err = m.Unlock()
	require.NoError(t, err)
}

func TestUnlockErrorsAreRecoverable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	err = m.Lock()
	require.NoError(t, err)

	// muck with the internal state in order to cause an error
	oldfd := m.fd
	m.fd = 99999

	// trigger an error
	err = m.Unlock()
	require.Error(t, err)

	// restore a sane internal state
	m.fd = oldfd

	// this would crash if we the mutex were unlocked in the error branch
	err = m.Unlock()
	require.NoError(t, err)
}

func TestRLockErrorsAreRecoverable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	// muck with the internal state in order to cause an error
	oldfd := m.fd
	m.fd = 99999

	// trigger an error
	err = m.RLock()
	require.Error(t, err)

	// restore a sane internal state
	m.fd = oldfd

	// this would hang if we hadn't Unlock()ed in the RLock error branch
	err = m.Lock()
	require.NoError(t, err)

	// clean up
	err = m.Unlock()
	require.NoError(t, err)
}

func TestRUnlockErrorsAreRecoverable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "x")
	m, err := New(path)
	require.NoError(t, err)

	err = m.RLock()
	require.NoError(t, err)

	// muck with the internal state in order to cause an error
	oldfd := m.fd
	m.fd = 99999

	// trigger an error
	err = m.RUnlock()
	require.Error(t, err)

	// restore a sane internal state
	m.fd = oldfd

	// this would crash if we the mutex were unlocked in the error branch
	err = m.RUnlock()
	require.NoError(t, err)
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
	assert.NoError(t, err)
}
