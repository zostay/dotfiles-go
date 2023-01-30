package mail

import (
	"os"
	"strings"
)

// MessageList is meant to streamline the reading of messages in a maildir
// folder. Reading all the messages in maildir folder is expensive, so let's
// just grab one at a time.
type MessageList interface {
	Next(msg *Message) bool
	Err() error
}

// DirFolderMessageList implements MessageList to iterate through all messages
// in a maildir folder.
type DirFolderMessageList struct {
	parent         *DirFolder
	remainingFiles map[string][]os.FileInfo
}

var rds = []string{"new", "cur"}

// currentReadDir returns the read directory we are working through along with
// the remaining unread files there. Returns ("", nil) if nothing is left to
// read.
func (ml *DirFolderMessageList) currentReadDir() (string, []os.FileInfo) {
	for _, rd := range rds {
		if fis, ok := ml.remainingFiles[rd]; ok {
			if len(fis) > 0 {
				return rd, fis
			}
			delete(ml.remainingFiles, rd)
		}
	}
	return "", nil
}

// nextFileInfo returns the read directory we are working through along with the
// next message in that directory. Returns ("", nil) if there's nothing left ot
// read. This dequeues the file in the process.
func (ml *DirFolderMessageList) nextFileInfo() (string, os.FileInfo) {
	rd, fis := ml.currentReadDir()
	if fis == nil {
		return "", nil
	}

	fi := fis[0]
	ml.remainingFiles[rd] = fis[1:]
	return rd, fi
}

// Next returns true if there is another message remaining to process. If true
// is returned, then the given message pointer will be overwritten with that
// message.
func (ml *DirFolderMessageList) Next(msg *Message) bool {
	for {
		rd, fi := ml.nextFileInfo()
		if fi == nil {
			return false
		}

		if strings.HasPrefix(fi.Name(), ".") {
			continue
		}

		m := ml.parent.message(rd, fi)
		*msg = *m // shallow copy

		return true
	}
}

// Err returns any error that may have occurred during iteration.
func (ml *DirFolderMessageList) Err() error {
	return nil
}

var _ MessageList = &DirFolderMessageList{}
