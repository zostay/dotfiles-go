package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// MailDirFolder represents a single maildir folder in a mail root.
type MailDirFolder struct {
	root     string
	basename string
}

// NewMailDirFolder constructs a MailDirFolder from a mail root and folder name.
func NewMailDirFolder(root, folder string) *MailDirFolder {
	return &MailDirFolder{root, folder}
}

// Root returns the path to the mail root.
func (f *MailDirFolder) Root() string { return f.root }

// Basename returns the folder name.
func (f *MailDirFolder) Basename() string { return f.basename }

// Path returns the full path to the maildir folder.
func (f *MailDirFolder) Path() string {
	return path.Join(f.root, f.basename)
}

// MessageDirPaths returns the directory paths that contain message files that
// can be worked with by these mail tools.
func (f *MailDirFolder) MessageDirPaths() []string {
	return []string{
		path.Join(f.root, f.basename, "new"),
		path.Join(f.root, f.basename, "cur"),
	}
}

// TempDirPath returns the directory path where messages being worked on are
// stored temporarily.
func (f *MailDirFolder) TempDirPath() string {
	return path.Join(f.root, f.basename, "tmp")
}

// message combines the common code of Message and Messages.
func (f *MailDirFolder) message(rd, fn string, fi os.FileInfo) *Message {
	var key, flags string
	if strings.ContainsRune(fi.Name(), ':') {
		parts := strings.SplitN(fi.Name(), ":", 2)
		key = parts[0]
		flags = parts[1]
	} else {
		key = fi.Name()
	}

	return NewMailDirMessageWithStat(key, flags, rd, f, &fi)
}

// Message returns a single mail message object stored in the given file name.
func (f *MailDirFolder) Message(fn string) (*Message, error) {
	for _, dir := range f.MessageDirPaths() {
		rd := path.Base(dir)
		msgPath := path.Join(dir, fn)
		fi, err := os.Stat(msgPath)
		if os.IsNotExist(err) {
			continue
		}

		return f.message(rd, fn, fi), nil
	}

	return nil, fmt.Errorf("no message named %q in folder %q", fn, f.Path())
}

// Messages returns all mail messages stored in the maildir.
func (f *MailDirFolder) Messages() ([]*Message, error) {
	var ms []*Message

	fism := make(map[string][]os.FileInfo)
	fiCount := 0
	for _, dir := range f.MessageDirPaths() {
		md, err := os.Open(dir)
		if err != nil {
			return ms, fmt.Errorf("unable to open maildir %q for reading: %w", f.basename, err)
		}

		fis, err := md.Readdir(0)
		if err != nil {
			return ms, fmt.Errorf("unable to read maildir %q file list: %w", f.basename, err)
		}

		rd := path.Base(dir)
		fism[rd] = fis
		fiCount += len(fis)
	}

	ms = make([]*Message, 0, fiCount)
	for rd, fis := range fism {
		for _, fi := range fis {
			if strings.HasPrefix(fi.Name(), ".") {
				continue
			}

			m := f.message(rd, fi.Name(), fi)
			ms = append(ms, m)
		}
	}

	return ms, nil
}
