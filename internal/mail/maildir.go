package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type MailDirFolder struct {
	root     string
	basename string
}

func NewMailDirFolder(root, folder string) *MailDirFolder {
	return &MailDirFolder{root, folder}
}

func (f *MailDirFolder) Root() string     { return f.root }
func (f *MailDirFolder) Basename() string { return f.basename }

func (f *MailDirFolder) Path() string {
	return path.Join(f.root, f.basename)
}

func (f *MailDirFolder) MessageDirPaths() []string {
	return []string{
		path.Join(f.root, f.basename, "new"),
		path.Join(f.root, f.basename, "cur"),
	}
}

func (f *MailDirFolder) TempDirPath() string {
	return path.Join(f.root, f.basename, "tmp")
}

func (f *MailDirFolder) Messages() ([]*Message, error) {
	var ms []*Message

	fism := make(map[string][]os.FileInfo)
	fiCount := 0
	for _, dir := range f.MessageDirPaths() {
		md, err := os.Open(dir)
		if err != nil {
			return ms, fmt.Errorf("unable to open maildir %s for reading: %w", f.basename, err)
		}

		fis, err := md.Readdir(0)
		if err != nil {
			return ms, fmt.Errorf("unable to read maildir %s file list: %w", f.basename, err)
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

			var key, flags string
			if strings.ContainsRune(fi.Name(), ':') {
				parts := strings.SplitN(fi.Name(), ":", 2)
				key = parts[0]
				flags = parts[1]
			} else {
				key = fi.Name()
			}

			m := NewMailDirMessageWithStat(key, flags, rd, f, &fi)
			ms = append(ms, m)
		}
	}

	return ms, nil
}
