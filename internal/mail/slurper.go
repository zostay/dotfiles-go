package mail

import (
	"io"
	"os"
	"path"
)

// Slurper describes the interface to implement for anything that can read an
// email message.
type Slurper interface {
	// Reader returns an io.Reader that will return the data.
	Reader() (io.Reader, error)

	// Filename gives the file name of the message on disk.
	Filename() string

	// Folder names the folder of the message on disk.
	Folder() string

	// Stat returns the file info for the message on disk or returns an error.
	Stat() (os.FileInfo, error)
}

// DirSlurper reads an email message in from a maildir folder.
type DirSlurper struct {
	key    string
	flags  string
	rd     string
	folder *DirFolder
	fi     *os.FileInfo
}

// NewMailDirSlurper creates and returns a *DirSlurper for the given folder,
// message key, message flags, and read status. No stat information is provided,
// so if Stat() is called, it will perform and cache that information later.
func NewMailDirSlurper(key, flags, rd string, folder *DirFolder) *DirSlurper {
	return &DirSlurper{key, flags, rd, folder, nil}
}

// NewMailDirSlurperWithStat is the same as NewMailDirSlurper, but it precaches
// the stat information.
func NewMailDirSlurperWithStat(key, flags, rd string, folder *DirFolder, fi *os.FileInfo) *DirSlurper {
	return &DirSlurper{key, flags, rd, folder, fi}
}

// Stat returns the cached os.FileInfo or performs an os.Stat() to read it in
// and return it it. It caches that information for next time.
func (r *DirSlurper) Stat() (os.FileInfo, error) {
	if r.fi != nil {
		return *r.fi, nil
	}

	fi, err := os.Stat(r.Filename())
	if err != nil {
		r.fi = &fi
	}
	return fi, err
}

// Slurp slurps up the file data and returns it.
func (r *DirSlurper) Reader() (io.Reader, error) {
	return os.Open(r.Filename())
}

// FlagSuffix returns the flags associated with this maildir message.
func (r *DirSlurper) FlagSuffix() string {
	if r.flags == "" {
		return ""
	}
	return ":" + r.flags
}

// Filename returns the full path to the maildir message, including the folder
// path, the read status folder it's in and the key and flag suffix.
func (r *DirSlurper) Filename() string {
	return path.Join(r.folder.Path(), r.rd, r.key+r.FlagSuffix())
}

// Folder returns the name of the folder the message is in.
func (r *DirSlurper) Folder() string {
	return r.folder.Basename()
}

// MoveTo moves the message to the target folder.
func (r *DirSlurper) MoveTo(target *DirFolder) error {
	targetFile := path.Join(target.Path(), r.rd, r.key+r.FlagSuffix())
	err := os.Rename(r.Filename(), targetFile)
	if err != nil {
		return err
	}

	r.folder = target

	return nil
}

// DirWriter perform message writing for a DirSlurper.
type DirWriter struct {
	r *DirSlurper

	tmp string
	f   io.WriteCloser
}

// NewMailDirWriter creates a DirWriter from the given DirSlurper. This will
// immediately open the file referred to by the *DirSlurper for overwriting.
func NewMailDirWriter(r *DirSlurper) (*DirWriter, error) {
	tmp := path.Join(r.folder.TempDirPath(), r.key+r.FlagSuffix())
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}

	w := DirWriter{r, tmp, f}
	return &w, nil
}

// Write saves the given bytes to disk.
func (w *DirWriter) Write(bs []byte) (int, error) {
	return w.f.Write(bs)
}

// Close closes open file handles.
func (w *DirWriter) Close() error {
	err := w.f.Close()
	if err != nil {
		return err
	}

	return os.Rename(w.tmp, w.r.Filename())
}

// Replace returns another *DirWriter for overwriting the current *DirWriter.
func (r *DirSlurper) Replace() (*DirWriter, error) {
	return NewMailDirWriter(r)
}

// Remove deletes the maildir message file.
func (r *DirSlurper) Remove() error {
	return os.Remove(r.Filename())
}

// MessageSlurper is able to read a MIME message from any file on disk.
type MessageSlurper struct {
	filename string
}

// NewMessageSlurper returns a *MessageSlurper for the given filename.
func NewMessageSlurper(filename string) *MessageSlurper {
	return &MessageSlurper{filename}
}

// Slurp reads in the MIME message.
func (r *MessageSlurper) Reader() (io.Reader, error) {
	return os.Open(r.filename)
}

// Filename returns the name of the message.
func (r *MessageSlurper) Filename() string {
	return r.filename
}

// Folder attempts to guess what the folder name is and returns that.
func (r *MessageSlurper) Folder() string {
	f := path.Base(r.filename)
	if dir := path.Dir(f); dir == "cur" || dir == "new" {
		f = path.Base(f)
	}
	return path.Dir(f)
}

// Stat uses os.Stat() to stat the file.
func (r *MessageSlurper) Stat() (os.FileInfo, error) {
	return os.Stat(r.filename)
}
