package mail

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/emersion/go-maildir"
	"gopkg.in/yaml.v3"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
)

var (
	labelBoxes = map[string]string{
		"\\Inbox":     "INBOX",
		"\\Trash":     "gmail.Trash",
		"\\Important": "gmail.Important",
		"\\Sent":      "gmail.Sent_Mail",
		"\\Starred":   "gmail.Starred",
		"\\Draft":     "gmail.Drafts",
	}
	boxLabels map[string]string

	DefaultMailDir = path.Join(dotfiles.HomeDir, "Mail")

	SkipFolder = map[string]struct{}{
		"gmail.Spam":      struct{}{},
		"gmail.Draft":     struct{}{},
		"gmail.Trash":     struct{}{},
		"gmail.Sent_mail": struct{}{},
	}
)

func init() {
	boxLabels = make(map[string]string, len(labelBoxes))
	for k, v := range labelBoxes {
		boxLabels[v] = k
	}
}

type Filter struct {
	MailRoot string
	Rules    CompiledRules
}

func NewFilter(root string) (*Filter, error) {
	f, err := LoadRules()
	if err != nil {
		return nil, err
	}

	return &Filter{
		MailRoot: root,
		Rules:    f,
	}, nil
}

func (fi *Filter) folder(folder string) maildir.Dir {
	return maildir.Dir(path.Join(fi.MailRoot, folder))
}

func (fi *Filter) Messages(folder string) ([]*Message, error) {
	var ms []*Message

	f := fi.folder(folder)

	ks, err := f.Keys()
	if err != nil {
		return ms, err
	}

	ms = make([]*Message, len(ks))
	for i, k := range ks {
		ms[i] = NewMessage(f, k)
	}

	return ms, nil
}

func (fi *Filter) Message(folder string, key string) *Message {
	f := fi.folder(folder)
	return NewMessage(f, key)
}

type ActionsSummary map[string]int

func (fi *Filter) LabelMessages() (ActionsSummary, error) {
	actions := make(ActionsSummary)
	folders := fi.Rules.FolderRules()

	for f, fr := range folders {
		err := fi.LabelFolderMessages(actions, f, fr)
		if err != nil {
			return actions, err
		}
	}

	return actions, nil
}

func (fi *Filter) LabelFolderMessages(
	actions ActionsSummary,
	folder string,
	rules CompiledRules,
) error {
	msgs, err := fi.Messages(folder)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		if _, skip := skipFolder[string(msg.folder)]; skip {
			continue
		}

		// Purged, leave it be
		has, err := msg.HasKeyword("\\Trash")
		if err != nil {
			return err
		} else if has {
			continue
		}

		for _, cr := range rules {
			as, err := msg.ApplyRule(cr)
			if err != nil {
				return err
			}

			for _, a := range as {
				actions[a]++
			}
		}
	}

	return nil
}

var (
	UnwantedFolderSuffix = []string{","}
	UnwantedFolderPrefix = []string{"+", "\\"}
	UnwantedFolder       = []string{"[", "]", "Drafts", "Home_School", "Network", "Pseudo-Junk.Social", "Pseudo-Junk.Social_Network", "Social Network", "OtherJunk"}
	UnwantedKeyword      = map[string][]string{
		"JunkSocial": {"Network", "Pseudo-Junk.Social", "Pseudo-Junk/Social", "Psuedo-Junk/Social_Network", "Pseudo-Junk.Social_Network"},
		"Teamwork":   {"Discussion"},
		"JunkOther":  {"OtherJunk"},
	}
)

func isUnwanted(folder string) bool {
	for _, us := range UnwantedFolderSuffix {
		if strings.HasSuffix(folder, us) {
			return true
		}
	}

	for _, up := range UnwantedFolderPrefix {
		if strings.HasPrefix(folder, up) {
			return true
		}
	}

	for _, uf := range UnwantedFolder {
		if folder == uf {
			return true
		}
	}

	return false
}

func hasUnwantedKeyword(msg *Message) ([]string, string, error) {
	for tok, uks := range UnwantedKeyword {
		for _, uk := range uks {
			unwanted, err := msg.HasKeyword(uk)
			if unwanted || err != nil {
				return uks, tok, err
			}
		}
	}

	return []string{}, "", nil
}

func (fi *Filter) Vacuum(logf func(fmt string, opts ...interface{})) error {
	md, err := os.Open(fi.MailRoot)
	if err != nil {
		return err
	}

	defer md.Close()

	folders, err := md.Readdir(0)
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if !folder.IsDir() {
			continue
		}

		if isUnwanted(folder) {
			logf("Droppping %s", folder.Name())

			msgs, err := fi.Messages(folder.Name())
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				other, err := msg.BestAlternateFolder()
				if err != nil {
					return err
				}

				msg.MoveTo(other)
				msg.RemoveKeyword(other)
				msg.Save()
				logf(" -> Moved %s to %s", folder.Name(), other)
			}

			deadFolder := path.Join(fi.MailRoot, folder.Name())
			for _, sd := range []string{"new", "cur", "tmp"} {
				err = os.Remove(path.Join(deadFolder, sd))
				if err != nil {
					logf("WARNING: cannot delete %s/%s: %+v", deadFolder, sd, err)
				}
			}
			err = os.Remove(deadFolder)
			if err != nil {
				logf("WARNING: cannot delete %s: %+v", deadFolder, err)
			}
		} else {
			logf("Searching %s for broken Keywords.", folder.Name())

			msgs, err := fi.Messages(folder.Name())
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				change := 0

				// Cleanup unwanted chars in keywords
				if msg.HasNonconformingKeywords {
					logf("Fixing non-conforming keywords.")
					msg.CleanupKeywords()
					change++
				}

				// Something went wrong somewhere
				unwanted, wanted, err := hasUnwantedKeyword(msg)
				if err != nil {
					return err
				}
				if unwanted != "" {
					logf("Fixing (%s) to %s.", strings.Join(unwanted, ", "), wanted)
					change++
					msg.RemoveKeyword(unwanted...)
					msg.AddKeyword(wanted)
				}

				if change > 0 {
					err := msg.Save()
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
