package filemutex

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
