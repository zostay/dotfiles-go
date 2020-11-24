package mail

import (
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/kr/pretty"
	"gopkg.in/yaml.v3"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
)

const (
	LocalLabelMailConf = ".label-mail.local.yml"
	LabelMailConf      = ".label-mail.yml"
)

type Match struct {
	Folder              string `yaml:"folder"`
	From                string `yaml:"from"`
	FromDomain          string `yaml:"from_domain"`
	To                  string `yaml:"to"`
	ToDomain            string `yaml:"to_domain"`
	Sender              string `yaml:"sender"`
	Subject             string `yaml:"subject"`
	SubjectFold         string `yaml:"isubject"`
	SubjectContains     string `yaml:"subject_contains"`
	SubjectContainsFold string `yaml:"subject_icontains"`
	Contains            string `yaml:"contains"`
	ContainsFold        string `yaml:"icontains"`
	Days                int    `yaml:"days"`
}

type CompiledRule struct {
	Match
	OkayDate time.Time

	Clear   []string
	Label   []string
	Move    string
	Forward []string
}

func (c *CompiledRule) IsClearing() bool   { return len(c.Clear) != 0 }
func (c *CompiledRule) IsLabeling() bool   { return len(c.Label) != 0 }
func (c *CompiledRule) IsMoving() bool     { return c.Move != "" }
func (c *CompiledRule) IsForwarding() bool { return len(c.Forward) != 0 }
func (c *CompiledRule) HasOkayDate() bool  { return c.OkayDate != time.Time{} }

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

type RawRule struct {
	Match
	Clear   interface{} `yaml:"clear"`
	Label   interface{} `yaml:"label"`
	Move    string      `yaml:"move"`
	Forward interface{} `yaml:"forward"`
}

type RawRules []RawRule
type EnvRawRules map[string]RawRules
type CompiledRules []*CompiledRule

func LoadRules() (CompiledRules, error) {
	var crs CompiledRules

	env, err := dotfiles.Environment()
	if err != nil {
		return crs, err
	}

	lbs, err := ioutil.ReadFile(path.Join(dotfiles.HomeDir, LabelMailConf))
	if err != nil {
		return crs, err
	}

	var pr EnvRawRules
	err = yaml.Unmarshal(lbs, &pr)
	if err != nil {
		return crs, err
	}

	llbs, err := ioutil.ReadFile(path.Join(dotfiles.HomeDir, LocalLabelMailConf))
	if err != nil {
		return crs, err
	}

	var lr RawRules
	err = yaml.Unmarshal(llbs, &lr)
	if err != nil {
		return crs, err
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

	crs = make(CompiledRules, 0, len(rr))
	for _, r := range rr {
		compiledLabel := CompileLabel("label", r.Label)
		compiledClear := CompileLabel("clear", r.Clear)

		compiledMove := strings.TrimSpace(r.Move)
		if compiledMove != "" {
			if ns, ok := labelBoxes[compiledMove]; ok {
				compiledMove = ns
			}
			compiledMove = strings.Replace(compiledMove, "/", ".", -1)
		}

		compiledForward := CompileField("forward", r.Forward)

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

func CompileField(name string, field interface{}) []string {
	var r1 []string
	if field == nil {
		r1 = []string{}
	} else {
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
	}

	return r1
}

func CompileLabel(name string, label interface{}) []string {
	r1 := CompileField(name, label)

	if len(r1) == 0 {
		return r1
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
		s = strings.Replace(s, ".", "/", -1)
		if ns, ok := boxLabels[s]; ok {
			s = ns
		}
		r2[i] = s
	}

	return r2
}

func (crs CompiledRules) FolderRules() CompiledFolderRules {
	fcrs := make(CompiledFolderRules)

	for _, cr := range crs {
		if cr.NeedsOkayDate() {
			days := 90
			if cr.Days != 0 {
				days = cr.Days
			}

			cr.OkayDate = time.Now().Add(time.Duration(-days) * time.Hour * 24)
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

type CompiledFolderRules map[string]CompiledRules

func (fcrs CompiledFolderRules) Add(folder string, cr *CompiledRule) {
	if fcr, ok := fcrs[folder]; ok {
		fcrs[folder] = append(fcr, cr)
	} else {
		fcrs[folder] = CompiledRules{cr}
	}
}
