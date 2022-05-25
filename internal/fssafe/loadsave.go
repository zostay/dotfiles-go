package fssafe

import (
	"io"
	"os"
)

// Loader is a function that returns a reader for a save file or database we
// want to safely read from.
type Loader func() (io.ReadCloser, error)

// Saver is a function that returns a writer to a save file or database we want
// to safely write to.
type Saver func() (io.WriteCloser, error)

// LoaderSaver is the interface that pairs a Loader with a Saver.
type LoaderSaver interface {
	Loader() (io.ReadCloser, error)
	Saver() (io.WriteCloser, error)
	LoaderFunc() Loader
	SaverFunc() Saver
}

// BasicLoaderSaver provides the minimum functionality for a LoaderSaver.
type BasicLoaderSaver struct {
	loader Loader // get a reader to load from
	saver  Saver  // get a writer to save to
}

// safeWriter is used internally to provide single file backups for a saved file
// by copying the file prior to writing to a path with .old suffixed to the
// file path. The writing of the file is written to a new file with .new
// suffixed to the end. When the writing is finished, the file is moved to the
// path properly. This makes sure that the file that is there for reading is
// complete and not in the midst of a write.
type safeWriter struct {
	w    *os.File
	path string
}

func (w *safeWriter) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *safeWriter) Close() error {
	err := w.w.Close()
	if err != nil {
		return err
	}

	_ = os.Rename(w.path, w.path+".old")
	err = os.Rename(w.path+".new", w.path)
	if err != nil {
		return err
	}

	return nil
}

// NewFileSystemLoaderSaver builds a file system loader/saver that creates
// single file backups prior to writing and attempts to make saves atomic by
// writing data to a temporary file, which is then moved in to replace the save
// file.
//
// This returns a LoaderSaver.
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

// Loader provides a io.ReadCloser for reading a save file.
func (ls *BasicLoaderSaver) Loader() (io.ReadCloser, error) {
	return ls.loader()
}

// LoaderFunc returns the Loader function used.
func (ls *BasicLoaderSaver) LoaderFunc() Loader {
	return ls.loader
}

// Saver provides an io.WriteCloser for writing a save file.
func (ls *BasicLoaderSaver) Saver() (io.WriteCloser, error) {
	return ls.saver()
}

// SaverFunc returns the Saver function used.
func (ls *BasicLoaderSaver) SaverFunc() Saver {
	return ls.saver
}
