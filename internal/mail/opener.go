package mail

import (
	"io"
	"os"
	"path"

	"github.com/emersion/go-maildir"
)

type Opener interface {
	Open() (io.ReadCloser, error)
	Filename() (string, error)
	Folder() string
}

type MailDirOpener struct {
	key    string
	folder maildir.Dir
}

func NewMailDirOpener(folder maildir.Dir, key string) *MailDirOpener {
	return &MailDirOpener{
		folder: folder,
		key:    key,
	}
}

func (r *MailDirOpener) Open() (io.ReadCloser, error) {
	return r.folder.Open(r.key)
}

func (r *MailDirOpener) Filename() (string, error) {
	return r.folder.Filename(r.key)
}

func (r *MailDirOpener) Folder() string {
	return path.Dir(string(r.folder))
}

func (r *MailDirOpener) MoveTo(target maildir.Dir) error {
	err := r.folder.Move(target, r.key)
	if err != nil {
		return err
	}

	r.folder = target

	return nil
}

func (r *MailDirOpener) Replace() (io.WriteCloser, error) {
	var w io.WriteCloser

	flags, err := r.folder.Flags(r.key)
	if err != nil {
		return w, err
	}

	key, w, err := r.folder.Create(flags)
	if err != nil {
		return w, err
	}

	r.key = key

	err = r.folder.Remove(r.key)
	if err != nil {
		return w, err
	}

	return w, nil
}

func (r *MailDirOpener) Remove() error {
	return r.folder.Remove(r.key)
}

type MessageOpener struct {
	filename string
}

func NewMessageOpener(filename string) *MessageOpener {
	return &MessageOpener{filename}
}

func (r *MessageOpener) Open() (io.ReadCloser, error) {
	return os.Open(r.filename)
}

func (r *MessageOpener) Filename() (string, error) {
	return r.filename, nil
}

func (r *MessageOpener) Folder() string {
	f := path.Base(r.filename)
	if dir := path.Dir(f); dir == "cur" || dir == "new" {
		f = path.Base(f)
	}
	return path.Dir(f)
}
