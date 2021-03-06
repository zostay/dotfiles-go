package mail

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
)

var (
	// labelBoxes is the map between Gmail's keywords and the special IMAP
	// folders synced by offlineimap
	labelBoxes = map[string]string{
		"\\Inbox":     "INBOX",
		"\\Trash":     "gmail.Trash",
		"\\Important": "gmail.Important",
		"\\Sent":      "gmail.Sent_Mail",
		"\\Starred":   "gmail.Starred",
		"\\Draft":     "gmail.Drafts",
	}

	// boxLabels is the inversion of labelBoxes
	boxLabels map[string]string

	// DefaultMailDir is the usual maildir
	DefaultMailDir = path.Join(dotfiles.HomeDir, "Mail")

	// SkipFolder lists folders that are never filtered.
	SkipFolder = map[string]struct{}{
		"gmail.Spam":      struct{}{},
		"gmail.Draft":     struct{}{},
		"gmail.Trash":     struct{}{},
		"gmail.Sent_Mail": struct{}{},
	}
)

func init() {
	boxLabels = make(map[string]string, len(labelBoxes))
	for k, v := range labelBoxes {
		boxLabels[v] = k
	}
}

// Filter represents the tools that parse and understand mail rules and filter
// folders and messages.
type Filter struct {
	MailRoot string        // maildir to filter
	Rules    CompiledRules // the compiled filter rules

	LimitRecent time.Duration // if set, only message files newer than this will be filtered

	Debug  int  // set the debug level, higher numbers mean even more verbose logging
	DryRun bool // when set, no changes will be made

	AllowSendingEmail bool // unless set, no email forwarding will be performed
}

// NewFilter loads the rules and prepares the system for message filtering.
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

// LimitFilterToRecent sets the LimitRecent time. When set and filtering
// folders, only messages with a modification time newer than LimitRecent will
// be filtered.
func (fi *Filter) LimitFilterToRecent(limit time.Duration) {
	fi.LimitRecent = limit
}

// LimitSince returns the LimitSince setting set by LimitFilterToRecent.
func (fi *Filter) LimitSince() time.Time {
	return time.Now().Add(-fi.LimitRecent)
}

// folder constructs a NewMailDirFolder for the named folder in the mail root.
func (fi *Filter) folder(folder string) *MailDirFolder {
	return NewMailDirFolder(fi.MailRoot, folder)
}

// Message returns a single message in a single folder.
func (fi *Filter) Message(folder, fn string) (*Message, error) {
	f := fi.folder(folder)
	m, err := f.Message(fn)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Messages returns all the messages that should be filtered in that folder.
func (fi *Filter) Messages(folder string) ([]*Message, error) {
	var ms []*Message

	f := fi.folder(folder)
	allms, err := f.Messages()
	if err != nil {
		return ms, err
	}

	ms = make([]*Message, 0, len(allms))

	var since time.Time
	if fi.LimitRecent > 0 {
		since = fi.LimitSince()
	}

	for _, m := range allms {
		if fi.LimitRecent > 0 {
			info, err := m.Stat()
			if err != nil {
				return ms, fmt.Errorf("unable to stat %q: %w", m.Filename(), err)
			}

			if info.ModTime().Before(since) {
				continue
			}
		}

		ms = append(ms, m)
	}

	return ms, nil
}

// AllFolders lists all the maildir folders in the mail root.
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

// RulesForFolder returns all the rules that apply to the given folder. The
// second return value is a boolean indicating whether this folder has any rules
// at all.
func (fi *Filter) RulesForFolder(f string) (CompiledRules, bool) {
	folders := fi.Rules.FolderRules()

	var (
		gr  CompiledRules
		gok bool
	)
	if gr, gok = folders[""]; !gok {
		gr = CompiledRules{}
	}

	var (
		fr CompiledRules
		ok bool
	)
	if fr, ok = folders[f]; ok || gok {
		if !ok {
			fr = make(CompiledRules, 0, len(gr))
		}

		fr = append(fr, gr...)

		return fr, true
	}

	return gr, gok
}

// LabelMessage applies filters to a specific message.
func (fi *Filter) LabelMessage(folder, fn string) (ActionsSummary, error) {
	actions := make(ActionsSummary)

	if fr, ok := fi.RulesForFolder(folder); ok {
		msg, err := fi.Message(folder, fn)
		if err != nil {
			return actions, err
		}

		as, err := fi.ApplyRules(msg, fr)
		if err != nil {
			return actions, err
		}

		for _, a := range as {
			actions[a]++
		}
	}

	return actions, nil
}

// LabelMessages applies filters to all applicable messages in the given list of
// folders.
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

	for _, f := range whichFolders {
		if fr, ok := fi.RulesForFolder(f); ok {
			err := fi.LabelFolderMessages(actions, f, fr)
			if err != nil {
				return actions, err
			}
		}
	}

	return actions, nil
}

// LabelFolderMessages performs filtering for a single maildir.
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

		if fi.Debug > 2 {
			cp.Fcolor(os.Stderr,
				"reading", "READING ",
				"file", fmt.Sprintf("%s\n", msg.Filename()),
			)
		}

		// Purged, leave it be
		has, err := msg.HasKeyword("\\Trash")
		if err != nil {
			return fmt.Errorf("error in %q: %v", msg.Filename(), err)
		} else if has {
			continue
		}

		as, err := fi.ApplyRules(msg, rules)
		if err != nil {
			return fmt.Errorf("error in %q: %v", msg.Filename(), err)
		}

		for _, a := range as {
			actions[a]++
		}
	}

	return nil
}

// ApplyRules applies all the rules to a single mail message.
func (fi *Filter) ApplyRules(msg *Message, rules []*CompiledRule) ([]string, error) {
	actions := make([]string, 0)
	for _, cr := range rules {
		as, err := fi.ApplyRule(msg, cr)
		if err != nil {
			return actions, err
		}

		actions = append(actions, as...)
	}

	return actions, nil
}

// ApplyRule applies a single mail filter rule to a single mail message.
func (fi *Filter) ApplyRule(m *Message, c *CompiledRule) ([]string, error) {
	var (
		fail    string
		passes  = make([]string, 0)
		actions []string
	)

	for _, skippable := range skipTests {
		r, err := skippable(m, c)

		if err != nil {
			cp.Fcolor(os.Stderr,
				"warn", "❗WARNING ",
				"meh", fmt.Sprintf(": %s. (", err),
				"file", m.Filename(),
				"meh", ")\n",
			)
		}

		if !r.skip {
			passes = append(passes, r.reason)
		} else {
			fail = r.reason
			break
		}
	}

	// if fail != "" {
	// 	return actions, nil
	// }

	tests := 0
	for _, applies := range ruleTests {
		r, err := applies(m, c, &tests)

		if err != nil {
			cp.Fcolor(os.Stderr,
				"warn", "❗WARNING ",
				"meh", fmt.Sprintf(": %s. (", err),
				"file", m.Filename(),
				"meh", ")\n",
			)
		}

		if r.pass {
			passes = append(passes, r.reason)
		} else {
			fail = r.reason
		}
	}

	// MOAR DEBUGGING
	if fi.Debug > 2 && fail != "" {
		cp.Fcolor(os.Stderr,
			"fail", "✗ FAILED",
			"meh", fmt.Sprintf(": %s.\n", fail),
		)
	}

	// AND EVEN MOAR DEBUGGING
	if fi.Debug > 2 || (fi.Debug > 1 && (len(passes) > 0 && fail == "")) {
		pass := "base"
		if fail == "" && tests > 0 {
			pass = "pass"
		}

		cp.Fcolor(os.Stderr,
			pass, "✔ PASSES",
			"base", fmt.Sprintf(": %s.\n", cp.Join("base", passes, ", ")),
		)
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
			f := m.r.Filename()
			cp.Fcolor(os.Stderr,
				strings.ToLower(op), op,
				"file", fmt.Sprintf(" %s ", f),
				"action", ": ",
				"value", fmt.Sprintf("%s\n", strings.Join(ts, ", ")),
			)
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
			err := m.ForwardTo(c.Forward)
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
