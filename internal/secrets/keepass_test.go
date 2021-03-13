package secrets

import (
	"bytes"
	"io"
	"io/ioutil"
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
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestKeepassLoaderSaver(t *testing.T) {
	var hasfile, hasnew, hasold bool

	tmpfile, err := ioutil.TempFile(os.TempDir(), "kbdx")
	if !assert.NoError(t, err, "able to get a tempfile") {
		return
	}

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

	s := RandString(20)
	tmpfile.WriteString(s)
	tmpfile.Close()

	k, err := newKeepass(fn, "testing", ZostayHighSecurityGroup)
	if !assert.NoError(t, err, "setup keepass") {
		return
	}

	r, err := k.loader()
	if !assert.NoError(t, err, "tempfile already exists, so reading it should be fine") {
		return
	}

	sr, err := ioutil.ReadAll(r)
	if !assert.NoError(t, err, "reading file should not have an err") {
		return
	}

	assert.Equal(t, s, sr, "found the same string from loader that we wrote")

	fi, err := os.Stat(fn)
	if !assert.NoError(t, err, "can stat the file") {
		return
	}
	size := fi.Size()
	mtime := fi.ModTime()

	w, err := k.saver()
	if !assert.NoError(t, err, "save creates file") {
		return
	}

	fi, err = os.Stat(fn + ".new")
	if !assert.NoError(t, err, "save created .new") {
		return
	}

	hasnew = true

	fi, err = os.Stat(fn)
	if !assert.NoError(t, err, "save preserved orig") {
		return
	}

	assert.Equal(t, size, fi.Size(), "orig size is same")
	assert.Equal(t, mtime, fi.ModTime(), "orig mtime is same")

	s = RandString(33)
	_, _ = io.WriteString(w, s)
	err = w.Close()
	if !assert.NoError(t, err, "close should create file") {
		return
	}

	fi, err = os.Stat(fn + ".new")
	if !assert.True(t, os.IsNotExist(err), ".new is gone") {
		return
	}

	newsize := fi.Size()
	newmtime := fi.ModTime()

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

type testReader struct {
	*bytes.Reader
	closed bool
}

func (t *testReader) Close() error { t.closed = true; return nil }

type testWriter struct {
	*bytes.Buffer
	closed bool
}

func (t *testWriter) Close() error { t.closed = true; return nil }

func TestKeepass(t *testing.T) {
	k, err := newKeepass("", "testing123", "Test")
	if !assert.NoError(t, err, "no error getting keepass") {
		return
	}

	var buf *bytes.Buffer

	rs := make([]*testReader, 0)
	k.loader = func() (io.ReadCloser, error) {
		r := &testReader{bytes.NewReader(buf.Bytes()), false}
		rs = append(rs, r)
		return r, nil
	}

	ws := make([]*testWriter, 0)
	k.saver = func() (io.WriteCloser, error) {
		buf = new(bytes.Buffer)
		w := &testWriter{buf, false}
		ws = append(ws, w)
		return w, nil
	}

	SecretKeeperTestSuite(k)

	for i, r := range rs {
		assert.Truef(t, r.closed, "reader %d was closed", i)
	}
	for i, w := range ws {
		assert.True(t, w.closed, "writer %d was closed", i)
	}
}
