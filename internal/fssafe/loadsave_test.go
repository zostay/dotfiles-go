package fssafe

import (
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))] //nolint:gosec // test does not need secure random
	}
	return string(b)
}

func TestLoaderSaver(t *testing.T) {
	t.Parallel()

	// get a tempfile to work with
	tmpfile, err := os.CreateTemp(os.TempDir(), "kbdx")
	if !assert.NoError(t, err, "able to get a tempfile") {
		return
	}

	// cleanup tooling
	var hasfile, hasnew, hasold bool
	hasfile = true
	fn := tmpfile.Name()
	defer func() {
		if hasfile {
			_ = os.Remove(fn)
		}
		if hasnew {
			_ = os.Remove(fn + ".new")
		}
		if hasold {
			_ = os.Remove(fn + ".old")
		}
	}()

	// writing starter data
	s := RandString(20)
	_, _ = tmpfile.WriteString(s)
	err = tmpfile.Close()
	if !assert.NoError(t, err, "closed initial tmpfile") {
		return
	}

	// setup
	k := NewFileSystemLoaderSaver(fn)

	// testing the loader
	r, err := k.Loader()
	if !assert.NoError(t, err, "tempfile already exists, so reading it should be fine") {
		return
	}

	sr, err := io.ReadAll(r)
	if !assert.NoError(t, err, "reading file should not have an err") {
		return
	}

	assert.Equal(t, []byte(s), sr, "found the same string from loader that we wrote")

	fi, err := os.Stat(fn)
	if !assert.NoError(t, err, "can stat the file") {
		return
	}
	size := fi.Size()
	mtime := fi.ModTime()

	// testing the saver
	w, err := k.Saver()
	if !assert.NoError(t, err, "save creates file") {
		return
	}

	_, err = os.Stat(fn + ".new")
	if !assert.NoError(t, err, "save created .new") {
		return
	}

	hasnew = true

	fi, err = os.Stat(fn)
	if !assert.NoError(t, err, "save preserved orig") {
		return
	}

	// backup file is okay
	assert.Equal(t, size, fi.Size(), "orig size is same")
	assert.Equal(t, mtime, fi.ModTime(), "orig mtime is same")

	s = RandString(33)
	_, _ = io.WriteString(w, s)

	err = w.(*safeWriter).w.Sync()
	if !assert.NoError(t, err, "sync worked") {
		return
	}

	fi, err = os.Stat(fn + ".new")
	if !assert.NoError(t, err, "stat new file while writing is ok") {
		return
	}

	newsize := fi.Size()
	newmtime := fi.ModTime()

	w.(*safeWriter).w, err = os.Open(fn)
	if !assert.NoError(t, err, "file still readable") {
		return
	}

	// saver close does some renaming
	err = w.Close()
	if !assert.NoError(t, err, "close should create file") {
		return
	}

	_, err = os.Stat(fn + ".new")
	if !assert.True(t, os.IsNotExist(err), ".new is gone") {
		return
	}

	fi, err = os.Stat(fn + ".old")
	if !assert.NoError(t, err, "orig is now .old") {
		return
	}

	hasold = true

	assert.Equal(t, size, fi.Size(), ".old is same size as old orig")
	assert.Equal(t, mtime, fi.ModTime(), ".old is same mtime as old orig")

	fi, err = os.Stat(fn)
	if !assert.NoError(t, err, ".new is now main") {
		return
	}

	assert.Equal(t, newsize, fi.Size(), "size matches what was .new")
	assert.Equal(t, newmtime, fi.ModTime(), "mtime matches what was .new")

	err = os.Remove(fn)
	if !assert.NoError(t, err, "failed to remove main") {
		return
	}

	hasfile = false

	_, err = k.loader()
	if !assert.Error(t, err, "tempfile was deleted, so loader should fail") {
		return
	}
}
