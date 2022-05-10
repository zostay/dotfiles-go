package mail

import (
	"io"
	"io/ioutil"
	"os"
	"path"
)

type Slurper interface {
	Slurp() ([]byte, error)
	Filename() string
	Folder() string
	Stat() (os.FileInfo, error)
}

type DirSlurper struct {
	key    string
	flags  string
	rd     string
	folder *DirFolder
	fi     *os.FileInfo
}

func NewMailDirSlurper(key, flags, rd string, folder *DirFolder) *DirSlurper {
	return &DirSlurper{key, flags, rd, folder, nil}
}

func NewMailDirSlurperWithStat(key, flags, rd string, folder *DirFolder, fi *os.FileInfo) *DirSlurper {
	return &DirSlurper{key, flags, rd, folder, fi}
}

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

func (r *DirSlurper) Slurp() ([]byte, error) {
	return ioutil.ReadFile(r.Filename())
}

func (r *DirSlurper) FlagSuffix() string {
	if r.flags == "" {
		return ""
	}
	return ":" + r.flags
}

func (r *DirSlurper) Filename() string {
	return path.Join(r.folder.Path(), r.rd, r.key+r.FlagSuffix())
}

func (r *DirSlurper) Folder() string {
	return r.folder.Basename()
}

func (r *DirSlurper) MoveTo(target *DirFolder) error {
	targetFile := path.Join(target.Path(), r.rd, r.key+r.FlagSuffix())
	err := os.Rename(r.Filename(), targetFile)
	if err != nil {
		return err
	}

	r.folder = target

	return nil
}

type DirWriter struct {
	r *DirSlurper

	tmp string
	f   io.WriteCloser
}

func NewMailDirWriter(r *DirSlurper) (*DirWriter, error) {
	tmp := path.Join(r.folder.TempDirPath(), r.key+r.FlagSuffix())
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}

	w := DirWriter{r, tmp, f}
	return &w, nil
}

func (w *DirWriter) Write(bs []byte) (int, error) {
	return w.f.Write(bs)
}

func (w *DirWriter) Close() error {
	err := w.f.Close()
	if err != nil {
		return err
	}

	return os.Rename(w.tmp, w.r.Filename())
}

func (r *DirSlurper) Replace() (*DirWriter, error) {
	return NewMailDirWriter(r)
}

func (r *DirSlurper) Remove() error {
	return os.Remove(r.Filename())
}

type MessageSlurper struct {
	filename string
}

func NewMessageSlurper(filename string) *MessageSlurper {
	return &MessageSlurper{filename}
}

func (r *MessageSlurper) Slurp() ([]byte, error) {
	return ioutil.ReadFile(r.filename)
}

func (r *MessageSlurper) Filename() string {
	return r.filename
}

func (r *MessageSlurper) Folder() string {
	f := path.Base(r.filename)
	if dir := path.Dir(f); dir == "cur" || dir == "new" {
		f = path.Base(f)
	}
	return path.Dir(f)
}

func (r *MessageSlurper) Stat() (os.FileInfo, error) {
	return os.Stat(r.filename)
}
