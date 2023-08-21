package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// DirFolder represents a single maildir folder in a mail root.
type DirFolder struct {
	root     string
	basename string
}

// NewMailDirFolder constructs a DirFolder from a mail root and folder name.
func NewMailDirFolder(root, folder string) *DirFolder {
	return &DirFolder{root, folder}
}

// EnsureExists creates the folder if it does not already exist.
func (f *DirFolder) EnsureExists() error {
	paths := []string{f.Path()}
	paths = append(paths, f.MessageDirPaths()...)
	paths = append(paths, f.TempDirPath())

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.MkdirAll(path, 0700)
			if err != nil {
				return fmt.Errorf("unable to create maildir %q (failed to create dir %q)q: %w", f.basename, path, err)
			}
		}
	}

	return nil
}

// Root returns the path to the mail root.
func (f *DirFolder) Root() string { return f.root }

// Basename returns the folder name.
func (f *DirFolder) Basename() string { return f.basename }

// Path returns the full path to the maildir folder.
func (f *DirFolder) Path() string {
	return path.Join(f.root, f.basename)
}

// MessageDirPaths returns the directory paths that contain message files that
// can be worked with by these mail tools.
func (f *DirFolder) MessageDirPaths() []string {
	return []string{
		path.Join(f.root, f.basename, "new"),
		path.Join(f.root, f.basename, "cur"),
	}
}

// TempDirPath returns the directory path where messages being worked on are
// stored temporarily.
func (f *DirFolder) TempDirPath() string {
	return path.Join(f.root, f.basename, "tmp")
}

// message combines the common code of Message and Messages.
func (f *DirFolder) message(rd string, fi os.FileInfo) *Message {
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
func (f *DirFolder) Message(fn string) (*Message, error) {
	for _, dir := range f.MessageDirPaths() {
		rd := path.Base(dir)
		msgPath := path.Join(dir, fn)
		fi, err := os.Stat(msgPath)
		if os.IsNotExist(err) {
			continue
		}

		return f.message(rd, fi), nil
	}

	return nil, fmt.Errorf("no message named %q in folder %q", fn, f.Path())
}

// Messages returns a MessageList, which can be used to efficiently iterate
// through all messages in a folder.
//
//	msgs, err := folder.Messages()
//	if err != nil {
//	  panic(err)
//	}
//	var msg Message
//	for msgs.Next(&msg) {
//	  # process msg ...
//	}
//	if err := msgs.Err(); err != nil {
//	  panic(err)
//	}
func (f *DirFolder) Messages() (*DirFolderMessageList, error) {
	fism := make(map[string][]os.FileInfo)
	fiCount := 0
	for _, dir := range f.MessageDirPaths() {
		md, err := os.Open(dir)
		if err != nil {
			return nil, fmt.Errorf("unable to open maildir %q for reading: %w", f.basename, err)
		}

		fis, err := md.Readdir(0)
		if err != nil {
			return nil, fmt.Errorf("unable to read maildir %q file list: %w", f.basename, err)
		}

		rd := path.Base(dir)
		fism[rd] = fis
		fiCount += len(fis)
	}

	return &DirFolderMessageList{
		parent:         f,
		remainingFiles: fism,
	}, nil
}
