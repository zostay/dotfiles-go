package mail

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/kr/pretty"
	"github.com/zostay/go-addr/pkg/addr"
	"gopkg.in/yaml.v3"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
)

const (
	// LocalLabelMailConf is the name of the local rules which are kept local on
	// disk and do not save in version control.
	LocalLabelMailConf = ".label-mail.local.yml"

	// LabelMailConf is the name of the generic rules which are kept in version
	// control.
	LabelMailConf = ".label-mail.yml"
)

// Match is the input configuration used for each rule.
type Match struct {
	// Folder is used to limit matching to an individual folder. If not given,
	// this rule will be applied to all folders.
	Folder string `yaml:"folder"`

	// From is used to match email addresses in the From header.
	From string `yaml:"from"`

	// FromDomain is used to match email address domains in the From header.
	FromDomain string `yaml:"from_domain"`

	// To is used to match email addresses in the To header.
	To string `yaml:"to"`

	// ToDomain is used to match email address domains in the To header.
	ToDomain string `yaml:"to_domain"`

	// Sender is used to match email addresses in the Sender header.
	Sender string `yaml:"sender"`

	// DeliveredTo is used to match email addresses in the Delivered-To header.
	DeliveredTo string `yaml:"delivered_to"`

	// Subject is used to match entire Subject header exactly.
	Subject string `yaml:"subject"`

	// SubjectFold is used to match entire Subject header exactly but with
	// case-insensitivity.
	SubjectFold string `yaml:"isubject"`

	// SubjectContains is used to match a substring of the Subject header.
	SubjectContains string `yaml:"subject_contains"`

	// SubjectContainsFold is used to match a substring of the Subject header,
	// but with case-insensitivity.
	SubjectContainsFold string `yaml:"subject_icontains"`

	// Contains is used ot match a substring anywhere in the email message.
	Contains string `yaml:"contains"`

	// ContainsFold is used to match a substring anywhere in the email message,
	// but with case-insensitivity.
	ContainsFold string `yaml:"icontains"`

	// Days limits matches to email messages older than the given number
	// of days.
	Days int `yaml:"days"`
}

// CompiledRule is the match after it has been processed by the rule compiler.
type CompiledRule struct {
	// Match is the original rule taken from the configuration file.
	Match

	// OkayDate is the date calculated from Days. A message does not match this
	// rule unless it has a Date header before the OkayDate.
	OkayDate time.Time

	// Clear lists the labels to clear from the message.
	Clear []string

	// Label lists the lables to add to the message.
	Label []string

	// Move lists the folder to move the message into.
	Move string

	// Forward gives the addresses to send the message to.
	Forward addr.AddressList
}

// IsClearing returns true if the message lists labels to clear.
func (c *CompiledRule) IsClearing() bool { return len(c.Clear) != 0 }

// IsLabeling returns true if the message lists labels to add.
func (c *CompiledRule) IsLabeling() bool { return len(c.Label) != 0 }

// IsMoving returns true if the message has a Move folder.
func (c *CompiledRule) IsMoving() bool { return c.Move != "" }

// IsForwarding returns true if the message has forwarding addresses.
func (c *CompiledRule) IsForwarding() bool { return len(c.Forward) != 0 }

// HasOkayDate returns true if the OkayDate is set.
func (c *CompiledRule) HasOkayDate() bool { return c.OkayDate != time.Time{} }

// NeedsOkayDate returns true if Days is set on the Match or if the rule adds
// the Trash label or if the rule moves the message to the Trash.
func (c *CompiledRule) NeedsOkayDate() bool {
	if c.Days != 0 {
		return true
	}

	if c.IsLabeling() {
		for _, l := range c.Label {
			if l == "\\Trash" {
				return true
			}
		}
	}

	if c.IsMoving() && c.Move == "gmail.Trash" {
		return true
	}

	return false
}

// RawRule is the rule in the configuration file combining both the Match and
// the actions to take.
type RawRule struct {
	// Match represents the matches to apply.
	Match `yaml:",inline"`

	// Clear is either a string or list containing labels to remove when a
	// message matches.
	Clear interface{} `yaml:"clear"`

	// Label is either a string or list containing labels to edd when a message
	// matches.
	Label interface{} `yaml:"label"`

	// Move is the name of the folder to move matching messages into.
	Move string `yaml:"move"`

	// Forward is the string or list containing email addresses to send the
	// message to if it matches.
	Forward interface{} `yaml:"forward"`
}

// RawRules is a list of rules
type RawRules []RawRule

// EnvRawRules is a list of rules sectioned by environment name.
type EnvRawRules map[string]RawRules

// CompiledRules is a list of compiled rules sectioned by environment name.
type CompiledRules []*CompiledRule

// LoadEnvRawRules loads the standard rules file split up into into environment
// sections.
func LoadEnvRawRules(rulePath string) (EnvRawRules, error) {
	var pr EnvRawRules

	lbs, err := os.ReadFile(rulePath)
	if err != nil {
		return pr, err
	}

	err = yaml.Unmarshal(lbs, &pr)
	return pr, err
}

// LoadRawRules loads the rules as a single section. (No sections split out by
// environment.)
func LoadRawRules(rulePath string) (RawRules, error) {
	var lr RawRules
	llbs, err := os.ReadFile(rulePath)
	if err != nil {
		return lr, err
	}

	err = yaml.Unmarshal(llbs, &lr)
	return lr, err
}

// DefaultPrimaryRulesConfigPath returns the default location for the primary
// rules file.
func DefaultPrimaryRulesConfigPath() string {
	return path.Join(dotfiles.HomeDir, LabelMailConf)
}

// DefaultLocalRulesConfigPath returns the default location for the local rules
// file.
func DefaultLocalRulesConfigPath() string {
	return path.Join(dotfiles.HomeDir, LocalLabelMailConf)
}

// LoadRules will load the rules from the various configuration files, combine,
// compile, and return them. Returns an error if something goes wrong.
//
// The primary file is the main configuration file with environment sections
// broken out (usually at located ~/.label-mail.yaml). The local file is the
// localized configuration file with no environment sections (usually located at
// ~/.label-mail.local.yaml).
func LoadRules(primary, local string) (CompiledRules, error) {
	env, err := dotfiles.Environment()
	if err != nil {
		return nil, err
	}

	pr, err := LoadEnvRawRules(primary)
	if err != nil {
		return nil, err
	}

	lr, err := LoadRawRules(local)
	if err != nil {
		return nil, err
	}

	ruleCount := len(lr)
	if _, ok := pr["*"]; ok {
		ruleCount += len(pr["*"])
	}
	if _, ok := pr[env]; ok {
		ruleCount += len(pr[env])
	}

	rr := make(RawRules, ruleCount)
	i := 0
	addRules := func(rs RawRules) {
		for _, r := range rs {
			rr[i] = r
			i++
		}
	}
	if rs, ok := pr["*"]; ok {
		addRules(rs)
	}
	if rs, ok := pr[env]; ok {
		addRules(rs)
	}
	addRules(lr)

	crs := make(CompiledRules, 0, len(rr))
	for _, r := range rr {
		compiledLabel := CompileLabel("label", r.Label)
		compiledClear := CompileLabel("clear", r.Clear)

		compiledMove := strings.TrimSpace(r.Move)
		if compiledMove != "" {
			if ns, ok := labelBoxes[compiledMove]; ok {
				compiledMove = ns
			}
			compiledMove = strings.ReplaceAll(compiledMove, "/", ".")
		}

		compiledForward, err := CompileAddress("forward", r.Forward)
		if err != nil {
			return crs, err
		}

		if len(compiledLabel) == 0 && len(compiledClear) == 0 && compiledMove == "" && len(compiledForward) == 0 {
			pretty.Printf("RULE MISSING ACTION %# v\n", r)
			continue
		}

		cr := CompiledRule{
			Match:   r.Match,
			Label:   compiledLabel,
			Clear:   compiledClear,
			Move:    compiledMove,
			Forward: compiledForward,
		}

		crs = append(crs, &cr)
	}

	return crs, nil
}

// CompileField handles fields that can either be provided as a list of items or
// a single string and turns them into a list of strings.
func CompileField(name string, field interface{}) []string {
	var r1 []string
	if field == nil {
		return nil
	}

	switch v := field.(type) {
	case string:
		r1 = []string{v}
	case []interface{}:
		r1 = make([]string, len(v))
		for i, vi := range v {
			switch vv := vi.(type) {
			case string:
				r1[i] = vv
			default:
				r1[i] = ""
				pretty.Printf("RULE HAS WEIRD %s: %# v\n", name, v)
			}
		}
	default:
		r1 = []string{}
		pretty.Printf("RULE HAS INCORRECT %s: %# v\n", name, v)
	}

	return r1
}

// CompileAddress handles fields that can either be provided as a list of items
// or a single string and turns them into an addr.AddressList. Returns an error
// if there's a problem parsing the email address(es).
func CompileAddress(name string, a interface{}) (addr.AddressList, error) {
	r1 := CompileField(name, a)
	if r1 == nil {
		return nil, nil
	}

	r2 := make(addr.AddressList, len(r1))
	for i, a := range r1 {
		var err error
		r2[i], err = addr.ParseEmailAddress(a)
		if err != nil {
			return r2, err
		}
	}
	return r2, nil
}

// CompileLabel provides special handling for label fields. It converts labels
// to their canonical form.
func CompileLabel(name string, label interface{}) []string {
	r1 := CompileField(name, label)

	if r1 == nil {
		return nil
	}

	r2 := make([]string, 0, len(r1))
	for _, s := range r1 {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		r2 = append(r2, s)
	}

	for i, s := range r2 {
		s = strings.ReplaceAll(s, ".", "/")
		if ns, ok := boxLabels[s]; ok {
			s = ns
		}
		r2[i] = s
	}

	return r2
}

// CompiledFolderRules are CompiledRules grouped by folder name.
type CompiledFolderRules map[string]CompiledRules

// FolderRules takes the compiled rules and groups them by folder. Any rule
// without a folder match will be added to every folder list. Those with a
// folder match will only be added to the folder with the same name. This
// performs some final cleanup on compiled rules as well.
func (crs CompiledRules) FolderRules(now time.Time) CompiledFolderRules {
	fcrs := make(CompiledFolderRules)

	for _, cr := range crs {
		if cr.NeedsOkayDate() {
			days := 90
			if cr.Days != 0 {
				days = cr.Days
			}

			cr.OkayDate = now.Add(time.Duration(-days) * time.Hour * 24)
		}

		folder := ""
		if cr.Folder != "" {
			folder = cr.Folder
		}

		fcrs.Add(folder, cr)

		if cr.IsMoving() && folder != "" {
			andClearInbox := cr
			andClearInbox.Move = ""
			andClearInbox.Folder = cr.Move
			andClearInbox.Clear = []string{"\\Inbox"}

			fcrs.Add(cr.Move, andClearInbox)
		}
	}

	return fcrs
}

// Add is a helper message that will cleanly append the rule to the
// CompiledRules in the named folder.
func (fcrs CompiledFolderRules) Add(folder string, cr *CompiledRule) {
	if fcr, ok := fcrs[folder]; ok {
		fcrs[folder] = append(fcr, cr)
	} else {
		fcrs[folder] = CompiledRules{cr}
	}
}
