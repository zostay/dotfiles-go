package mail

import (
	"io"
	"os"
	"path"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Opener interface {
	Open() (ReadSeekCloser, error)
	Filename() string
	Folder() string
	Stat() (os.FileInfo, error)
}

type MailDirOpener struct {
	key    string
	flags  string
	rd     string
	folder *MailDirFolder
	fi     *os.FileInfo
}

func NewMailDirOpener(key, flags, rd string, folder *MailDirFolder) *MailDirOpener {
	return &MailDirOpener{key, flags, rd, folder, nil}
}

func NewMailDirOpenerWithStat(key, flags, rd string, folder *MailDirFolder, fi *os.FileInfo) *MailDirOpener {
	return &MailDirOpener{key, flags, rd, folder, fi}
}

func (r *MailDirOpener) Stat() (os.FileInfo, error) {
	if r.fi != nil {
		return *r.fi, nil
	}

	fi, err := os.Stat(r.Filename())
	if err != nil {
		r.fi = &fi
	}
	return fi, err
}

func (r *MailDirOpener) Open() (ReadSeekCloser, error) {
	return os.Open(r.Filename())
}

func (r *MailDirOpener) FlagSuffix() string {
	if r.flags == "" {
		return ""
	}
	return ":" + r.flags
}

func (r *MailDirOpener) Filename() string {
	return path.Join(r.folder.Path(), r.rd, r.key+r.FlagSuffix())
}

func (r *MailDirOpener) Folder() string {
	return r.folder.Basename()
}

func (r *MailDirOpener) MoveTo(target *MailDirFolder) error {
	targetFile := path.Join(target.Path(), r.rd, r.key+r.FlagSuffix())
	err := os.Rename(r.Filename(), targetFile)
	if err != nil {
		return err
	}

	r.folder = target

	return nil
}

type MailDirWriter struct {
	r *MailDirOpener

	tmp string
	f   io.WriteCloser
}

func NewMailDirWriter(r *MailDirOpener) (*MailDirWriter, error) {
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

func (r *MailDirOpener) Replace() (*MailDirWriter, error) {
	return NewMailDirWriter(r)
}

func (r *MailDirOpener) Remove() error {
	return os.Remove(r.Filename())
}

type MessageOpener struct {
	filename string
}

func NewMessageOpener(filename string) *MessageOpener {
	return &MessageOpener{filename}
}

func (r *MessageOpener) Open() (ReadSeekCloser, error) {
	return os.Open(r.filename)
}

func (r *MessageOpener) Filename() string {
	return r.filename
}

func (r *MessageOpener) Folder() string {
	f := path.Base(r.filename)
	if dir := path.Dir(f); dir == "cur" || dir == "new" {
		f = path.Base(f)
	}
	return path.Dir(f)
}

func (r *MessageOpener) Stat() (os.FileInfo, error) {
	return os.Stat(r.filename)
}
