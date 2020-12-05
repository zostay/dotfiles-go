package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/emersion/go-maildir"

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

	LimitRecent time.Duration

	Debug  int
	DryRun bool

	AllowSendingEmail bool
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

func (fi *Filter) LimitFilterToRecent(limit time.Duration) {
	fi.LimitRecent = limit
}

func (fi *Filter) LimitSince() time.Time {
	return time.Now().Add(-fi.LimitRecent)
}

func (fi *Filter) folder(folder string) maildir.Dir {
	return maildir.Dir(path.Join(fi.MailRoot, folder))
}

func (fi *Filter) Messages(folder string) ([]*Message, error) {
	var ms []*Message

	f := fi.folder(folder)

	ks, err := f.Keys()
	if err != nil {
		return ms, fmt.Errorf("unable to retrieve keys from maildir %s: %w", f, err)
	}

	var since time.Time
	if fi.LimitRecent > 0 {
		since = fi.LimitSince()
	}

	ms = make([]*Message, 0, len(ks))
	for _, k := range ks {
		if fi.LimitRecent > 0 {
			fn, err := f.Filename(k)
			if err != nil {
				return ms, fmt.Errorf("unable to get filename for folder %s and key %s: %w", folder, k, err)
			}

			info, err := os.Stat(fn)
			if err != nil {
				return ms, fmt.Errorf("unable to stat %s: %w", fn, err)
			}

			if info.ModTime().Before(since) {
				continue
			}
		}

		ms = append(ms, NewMailDirMessage(f, k))
	}

	return ms, nil
}

func (fi *Filter) Message(folder string, key string) *Message {
	f := fi.folder(folder)
	return NewMailDirMessage(f, key)
}

type ActionsSummary map[string]int

func (fi *Filter) AllFolders() ([]string, error) {
	var folderNames []string

	md, err := os.Open(fi.MailRoot)
	if err != nil {
		return folderNames, err
	}

	defer md.Close()

	folders, err := md.Readdir(0)
	if err != nil {
		return folderNames, err
	}

	folderNames = make([]string, 0, len(folders))
	for _, folder := range folders {
		if !folder.IsDir() {
			continue
		}

		folderNames = append(folderNames, folder.Name())
	}

	return folderNames, nil
}

func (fi *Filter) LabelMessages(onlyFolders []string) (ActionsSummary, error) {
	actions := make(ActionsSummary)

	var whichFolders []string
	if len(onlyFolders) == 0 {
		var err error
		whichFolders, err = fi.AllFolders()
		if err != nil {
			return actions, err
		}
	} else {
		whichFolders = onlyFolders
	}

	folders := fi.Rules.FolderRules()

	var (
		gr  CompiledRules
		gok bool
	)
	if gr, gok = folders[""]; !gok {
		gr = CompiledRules{}
	}

	for _, f := range whichFolders {
		var (
			fr CompiledRules
			ok bool
		)
		if fr, ok = folders[f]; ok || gok {
			if !ok {
				fr = make(CompiledRules, 0, len(gr))
			}

			fr = append(fr, gr...)

			err := fi.LabelFolderMessages(actions, f, fr)
			if err != nil {
				return actions, err
			}
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
		if _, skip := SkipFolder[msg.r.Folder()]; skip {
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
			as, err := fi.ApplyRule(msg, cr)
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
	folders, err := fi.AllFolders()
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if isUnwanted(folder) {
			logf("Droppping %s", folder)

			msgs, err := fi.Messages(folder)
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				other, err := msg.BestAlternateFolder()
				if err != nil {
					return err
				}

				err = msg.MoveTo(fi.MailRoot, other)
				if err != nil {
					return err
				}

				err = msg.RemoveKeyword(other)
				if err != nil {
					return err
				}

				err = msg.Save()
				if err != nil {
					return err
				}

				logf(" -> Moved %s to %s", folder, other)
			}

			deadFolder := path.Join(fi.MailRoot, folder)
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
			logf("Searching %s for broken Keywords.", folder)

			msgs, err := fi.Messages(folder)
			if err != nil {
				return err
			}

			for _, msg := range msgs {
				change := 0

				// Cleanup unwanted chars in keywords
				nonconforming, err := msg.HasNonconformingKeywords()
				if err != nil {
					return err
				}
				if nonconforming {
					logf("Fixing non-conforming keywords.")
					err := msg.CleanupKeywords()
					if err != nil {
						return err
					}
					change++
				}

				// Something went wrong somewhere
				unwanted, wanted, err := hasUnwantedKeyword(msg)
				if err != nil {
					return err
				}
				if len(unwanted) > 0 {
					logf("Fixing (%s) to %s.", strings.Join(unwanted, ", "), wanted)
					change++
					err := msg.RemoveKeyword(unwanted...)
					if err != nil {
						return err
					}

					err = msg.AddKeyword(wanted)
					if err != nil {
						return err
					}
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

func (fi *Filter) ApplyRule(m *Message, c *CompiledRule) ([]string, error) {
	var (
		fail    string
		passes  = make([]string, 0)
		actions []string
	)

	for _, skippable := range skipTests {
		r, err := skippable(m, c)

		if err != nil {
			return actions, err
		}

		if !r.skip {
			passes = append(passes, r.reason)
		} else {
			fail = r.reason
			break
		}
	}

	if fail != "" {
		return actions, nil
	}

	tests := 0
	for _, applies := range ruleTests {
		r, err := applies(m, c, &tests)

		if err != nil {
			return actions, err
		}

		if r.pass {
			passes = append(passes, r.reason)
		} else {
			fail = r.reason
		}
	}

	// MOAR DEBUGGING
	if fi.Debug > 2 && fail != "" {
		fmt.Fprintf(os.Stderr, "FAILED: %s.\n", fail)
	}

	// AND EVEN MOAR DEBUGGING
	if fi.Debug > 2 || (fi.Debug > 1 && (len(passes) > 0 && fail == "")) {
		fmt.Fprintf(os.Stderr, "PASSES: %s.\n", strings.Join(passes, ", "))
	}

	if fail != "" {
		return actions, nil
	}

	if tests == 0 {
		return actions, nil
	}

	actions = make([]string, 0, 1)

	debugLogOp := func(op string, m *Message, ts []string) {
		if fi.Debug > 0 {
			f, _ := m.r.Filename()
			fmt.Fprintf(os.Stderr, "%s %s : %s\n", op, f, strings.Join(ts, ", "))
		}
	}

	if c.IsLabeling() {
		if !fi.DryRun {
			err := m.AddKeyword(c.Label...)
			if err != nil {
				return actions, err
			}
		}

		debugLogOp("LABELING", m, c.Label)

		actions = append(actions, "Labeled "+strings.Join(c.Label, ", "))
	}

	if c.IsClearing() {
		if !fi.DryRun {
			err := m.RemoveKeyword(c.Clear...)
			if err != nil {
				return actions, err
			}
		}

		debugLogOp("CLEARING", m, c.Clear)

		actions = append(actions, "Cleared "+strings.Join(c.Clear, ", "))
	}

	if c.IsForwarding() {
		if !fi.DryRun && fi.AllowSendingEmail {
			err := m.ForwardTo(c.Forward...)
			if err != nil {
				return actions, err
			}
		}

		debugLogOp("FORWARDING", m, AddressListStrings(c.Forward))

		if fi.AllowSendingEmail {
			actions = append(actions, "Forwarded "+strings.Join(AddressListStrings(c.Forward), ", "))
		} else {
			actions = append(actions, "NOT Forwarded "+strings.Join(AddressListStrings(c.Forward), ", "))
		}
	}

	if len(actions) > 0 && !fi.DryRun {
		err := m.Save()
		if err != nil {
			return actions, err
		}
	}

	if c.IsMoving() {
		if !fi.DryRun {
			err := m.MoveTo(fi.MailRoot, c.Move)
			if err != nil {
				return actions, err
			}
		}

		debugLogOp("MOVING", m, []string{c.Move})

		actions = append(actions, "Moved "+c.Move)
	}

	return actions, nil
}
