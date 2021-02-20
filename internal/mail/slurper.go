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

type MailDirSlurper struct {
	key    string
	flags  string
	rd     string
	folder *MailDirFolder
	fi     *os.FileInfo
}

func NewMailDirSlurper(key, flags, rd string, folder *MailDirFolder) *MailDirSlurper {
	return &MailDirSlurper{key, flags, rd, folder, nil}
}

func NewMailDirSlurperWithStat(key, flags, rd string, folder *MailDirFolder, fi *os.FileInfo) *MailDirSlurper {
	return &MailDirSlurper{key, flags, rd, folder, fi}
}

func (r *MailDirSlurper) Stat() (os.FileInfo, error) {
	if r.fi != nil {
		return *r.fi, nil
	}

	fi, err := os.Stat(r.Filename())
	if err != nil {
		r.fi = &fi
	}
	return fi, err
}

func (r *MailDirSlurper) Slurp() ([]byte, error) {
	return ioutil.ReadFile(r.Filename())
}

func (r *MailDirSlurper) FlagSuffix() string {
	if r.flags == "" {
		return ""
	}
	return ":" + r.flags
}

func (r *MailDirSlurper) Filename() string {
	return path.Join(r.folder.Path(), r.rd, r.key+r.FlagSuffix())
}

func (r *MailDirSlurper) Folder() string {
	return r.folder.Basename()
}

func (r *MailDirSlurper) MoveTo(target *MailDirFolder) error {
	targetFile := path.Join(target.Path(), r.rd, r.key+r.FlagSuffix())
	err := os.Rename(r.Filename(), targetFile)
	if err != nil {
		return err
	}

	r.folder = target

	return nil
}

type MailDirWriter struct {
	r *MailDirSlurper

	tmp string
	f   io.WriteCloser
}

func NewMailDirWriter(r *MailDirSlurper) (*MailDirWriter, error) {
	tmp := path.Join(r.folder.TempDirPath(), r.key+r.FlagSuffix())
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}

	w := MailDirWriter{r, tmp, f}
	return &w, nil
}

func (w *MailDirWriter) Write(bs []byte) (int, error) {
	return w.f.Write(bs)
}

func (w *MailDirWriter) Close() error {
	err := w.f.Close()
	if err != nil {
		return err
	}

	return os.Rename(w.tmp, w.r.Filename())
}

func (r *MailDirSlurper) Replace() (*MailDirWriter, error) {
	return NewMailDirWriter(r)
}

func (r *MailDirSlurper) Remove() error {
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
