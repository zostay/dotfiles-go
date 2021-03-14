package fssafe

import (
	"bytes"
	"io"
)

type TestingReader struct {
	*bytes.Reader
	Closed bool
}

func (t *TestingReader) Close() error { t.Closed = true; return nil }

type TestingWriter struct {
	*bytes.Buffer
	Closed bool
}

func (t *TestingWriter) Close() error { t.Closed = true; return nil }

type TestingLoaderSaver struct {
	BasicLoaderSaver
	Readers []*TestingReader
	Writers []*TestingWriter
}

func NewTestingLoaderSaver() *TestingLoaderSaver {
	rs := make([]*TestingReader, 0)
	ws := make([]*TestingWriter, 0)

	var buf *bytes.Buffer
	loader := func() (io.ReadCloser, error) {
		r := &TestingReader{bytes.NewReader(buf.Bytes()), false}
		rs = append(rs, r)
		return r, nil
	}

	saver := func() (io.WriteCloser, error) {
		buf = new(bytes.Buffer)
		w := &TestingWriter{buf, false}
		ws = append(ws, w)
		return w, nil
	}

	ls := TestingLoaderSaver{BasicLoaderSaver{loader, saver}, rs, ws}

	return &ls
}
