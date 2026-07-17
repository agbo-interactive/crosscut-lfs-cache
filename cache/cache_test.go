package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := NewFilesystemCache(dir)
	require.NoError(t, err)

	write := func(key string) error {
		cr, cw, source, err := c.Get(key, 6)
		if err != nil {
			return err
		}

		if source != SourceFresh {
			return fmt.Errorf("expected source to be %v, got %v", SourceFresh, source)
		}

		if _, err := cw.Write([]byte("foobar")); err != nil {
			return err
		}

		return cr.Close()
	}

	read := func(key string, expectedSource Source) error {
		cr, cw, source, err := c.Get(key, 6)
		if err != nil {
			return err
		}

		if cw != nil {
			return fmt.Errorf("expected writer to be nil")
		}

		if source != expectedSource {
			return fmt.Errorf("expected source to be %v, got %v", expectedSource, source)
		}

		p := make([]byte, 6)
		n, err := cr.Read(p)
		if err != nil {
			return err
		}

		if n != len(p) {
			return fmt.Errorf("expected length to be %d, got %d", len(p), n)
		}

		return cr.Close()
	}

	require.NoError(t, write("foobar"))
	require.NoError(t, read("foobar", SourceInflight))
	require.NoError(t, c.Done("foobar", nil))
	require.FileExists(t, filepath.Join(dir, DirObjects, DefaultFilenamer("foobar")))

	require.EqualError(t, c.Done("foobar", nil), ErrKeyNotFound.Error())

	require.NoError(t, read("foobar", SourceDisk))
	require.NoError(t, write("hello"))
	require.NoError(t, c.Done("hello", fmt.Errorf("fake error")))

	_, err = os.Stat(filepath.Join(dir, DirObjects, DefaultFilenamer("hello")))
	require.Error(t, err)
}

func TestCacheDiscardsWrongSizeEntry(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	c, err := NewFilesystemCache(dir)
	require.NoError(t, err)

	var handlerKey string
	var handlerExpected, handlerActual int64
	c.CorruptEntryHandler = func(key string, expectedSize, actualSize int64) {
		handlerKey, handlerExpected, handlerActual = key, expectedSize, actualSize
	}

	// populate an entry, then truncate it on disk to simulate storage corruption
	cr, cw, source, err := c.Get("foobar", 6)
	require.NoError(t, err)
	require.Equal(t, SourceFresh, source)
	_, err = cw.Write([]byte("foobar"))
	require.NoError(t, err)
	require.NoError(t, cr.Close())
	require.NoError(t, c.Done("foobar", nil))

	filename := filepath.Join(dir, DirObjects, DefaultFilenamer("foobar"))
	require.NoError(t, ioutil.WriteFile(filename, []byte("foo"), 0600))

	// the corrupt entry is discarded and Get behaves as a cache miss
	cr, cw, source, err = c.Get("foobar", 6)
	require.NoError(t, err)
	require.Equal(t, SourceFresh, source)
	require.NotNil(t, cw)
	require.NoError(t, cr.Close())
	require.NoError(t, c.Done("foobar", fmt.Errorf("abort refill")))

	_, err = os.Stat(filename)
	require.Error(t, err)

	require.Equal(t, "foobar", handlerKey)
	require.Equal(t, int64(6), handlerExpected)
	require.Equal(t, int64(3), handlerActual)

	// a negative expected size skips the check
	require.NoError(t, ioutil.WriteFile(filename, []byte("foo"), 0600))
	cr, cw, source, err = c.Get("foobar", -1)
	require.NoError(t, err)
	require.Equal(t, SourceDisk, source)
	require.Nil(t, cw)
	require.NoError(t, cr.Close())
}
