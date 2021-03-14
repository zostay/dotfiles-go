package fssafe

import (
	"io"
	"os"
)

type Loader func() (io.ReadCloser, error)
type Saver func() (io.WriteCloser, error)

type LoaderSaver interface {
	Loader() (io.ReadCloser, error)
	Saver() (io.WriteCloser, error)
	LoaderFunc() Loader
	SaverFunc() Saver
}

type BasicLoaderSaver struct {
	loader Loader // get a reader to load from
	saver  Saver  // get a writer to save to
}

type safeWriter struct {
	w    *os.File
	path string
}

func (w *safeWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *safeWriter) Close() error {
	w.w.Close()

	_ = os.Rename(w.path, w.path+".old")
	err := os.Rename(w.path+".new", w.path)
	if err != nil {
		return err
	}

	return nil
}

func NewFileSystemLoaderSaver(path string) *BasicLoaderSaver {
	loader := func() (io.ReadCloser, error) {
		dfr, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		return dfr, nil
	}

	saver := func() (io.WriteCloser, error) {
		cfw, err := os.Create(path + ".new")
		if err != nil {
			return nil, err
		}

		return &safeWriter{cfw, path}, nil
	}

	return &BasicLoaderSaver{loader, saver}
}

func (ls *BasicLoaderSaver) Loader() (io.ReadCloser, error) {
	return ls.loader()
}

func (ls *BasicLoaderSaver) LoaderFunc() Loader {
	return ls.loader
}

func (ls *BasicLoaderSaver) Saver() (io.WriteCloser, error) {
	return ls.saver()
}

func (ls *BasicLoaderSaver) SaverFunc() Saver {
	return ls.saver
}
