package mail

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/zostay/go-addr/pkg/addr"
	"github.com/zostay/go-email/pkg/email/mime"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
	"github.com/zostay/dotfiles-go/internal/xtrings"
)

const (
	FromName = "Andrew Sterling Hanenkamp"
)

var (
	FromEmail = dotfiles.MustGetSecret("GIT_EMAIL_HOME")

	SASLUser = dotfiles.MustGetSecret("LABEL_MAIL_USERNAME")
	SASLPass = dotfiles.MustGetSecret("LABEL_MAIL_PASSWORD")
)

var (
	FromEmailAddress AddressList
)

func init() {
	FromEmailAddress = make(AddressList, 1)
	FromEmailAddress[0] = &Address{
		Name:    FromName,
		Address: FromEmail,
	}
}

type Message struct {
	r Slurper
	m *mime.Message
}

func NewMessage(r Slurper) *Message {
	return &Message{r: r}
}

func NewMailDirMessage(key, flags, rd string, folder *MailDirFolder) *Message {
	r := NewMailDirSlurper(key, flags, rd, folder)
	return NewMessage(r)
}

func NewMailDirMessageWithStat(key, flags, rd string, folder *MailDirFolder, fi *os.FileInfo) *Message {
	r := NewMailDirSlurperWithStat(key, flags, rd, folder, fi)
	return NewMessage(r)
}

func NewFileMessage(filename string) *Message {
	r := NewMessageSlurper(filename)
	return NewMessage(r)
}

func (m *Message) Filename() string {
	return m.r.Filename()
}

func (m *Message) Stat() (os.FileInfo, error) {
	return m.r.Stat()
}

func (m *Message) EmailMessage() (*mime.Message, error) {
	if m.m != nil {
		return m.m, nil
	}

	bs, err := m.r.Slurp()
	if err != nil {
		return nil, err
	}

	m.m, err = mime.Parse(bs)
	return m.m, err
}

func (m *Message) Raw() ([]byte, error) {
	return m.r.Slurp()
}

func (m *Message) Date() (time.Time, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return time.Time{}, err
	}

	return mm.HeaderDate()
}

func (m *Message) Keywords() ([]string, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	sk := mm.HeaderGet("Keywords")
	ks := strings.FieldsFunc(sk, func(c rune) bool {
		return unicode.IsSpace(c) || c == ','
	})

	sort.Strings(ks)

	return ks, nil
}

func (m *Message) KeywordsSet() (km map[string]struct{}, err error) {
	ks, err := m.Keywords()
	if err != nil {
		return
	}

	km = make(map[string]struct{}, len(ks))
	for _, k := range ks {
		km[k] = struct{}{}
	}

	return
}

func (m *Message) HasNonconformingKeywords() (bool, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return false, err
	}

	sk := mm.HeaderGet("Keywords")
	where := strings.IndexFunc(sk, func(c rune) bool {
		return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '-' || c == '.' || c == '/'
	})

	return where >= 0, nil
}

func (m *Message) HasKeyword(names ...string) (bool, error) {
	km, err := m.KeywordsSet()
	if err != nil {
		return false, err
	}

	for _, n := range names {
		if _, ok := km[n]; !ok {
			return false, nil
		}
	}

	return true, nil
}

func (m *Message) MissingKeyword(names ...string) (bool, error) {
	km, err := m.KeywordsSet()
	if err != nil {
		return false, err
	}

	for _, n := range names {
		if _, ok := km[n]; ok {
			return false, nil
		}
	}

	return true, nil
}

func (m *Message) CleanupKeywords() error {
	ks, err := m.Keywords()
	if err != nil {
		return err
	}

	h, err := m.EmailMessage()
	if err != nil {
		return err
	}

	err = h.HeaderSet("Keywords", strings.Join(ks, " "))
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) AddKeyword(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	km, err := m.KeywordsSet()
	if err != nil {
		return err
	}

	for _, n := range names {
		if k, ok := boxLabels[n]; ok {
			km[k] = struct{}{}
		} else {
			km[n] = struct{}{}
		}
	}

	return m.updateKeywords(km)
}

func (m *Message) updateKeywords(km map[string]struct{}) error {
	mm, err := m.EmailMessage()
	if err != nil {
		return err
	}

	ks := make([]string, 0, len(km))
	for k := range km {
		ks = append(ks, k)
	}

	sort.Strings(ks)

	err = mm.HeaderSet("Keywords", strings.Join(ks, ", "))
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) RemoveKeyword(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	km, err := m.KeywordsSet()
	if err != nil {
		return err
	}

	for _, n := range names {
		if k, ok := boxLabels[n]; ok {
			delete(km, k)
		} else {
			delete(km, n)
		}
	}

	return m.updateKeywords(km)
}

func (m *Message) AddressList(key string) (addr.AddressList, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	return mm.HeaderGetAddressList(key)
}

func (m *Message) Subject() (string, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return "", err
	}

	return mm.HeaderGet("Subject"), nil
}

func (m *Message) Folder() (string, error) {
	return m.r.Folder(), nil
}

type skipTest func(*Message, *CompiledRule) (skipResult, error)
type ruleTest func(*Message, *CompiledRule, *int) (testResult, error)

type skipResult struct {
	skip   bool
	reason string
}

type testResult struct {
	pass   bool
	reason string
}

var (
	skipTests = []skipTest{
		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsLabeling() {
				return skipResult{false, "not labeling"}, nil
			}

			ok, err := m.HasKeyword(c.Label...)
			if !ok {
				return skipResult{false, fmt.Sprintf("needs labels [%s]", strings.Join(c.Label, ", "))}, err
			}

			return skipResult{true, fmt.Sprintf("already labeled [%s]", strings.Join(c.Label, ", "))}, err
		},

		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsClearing() {
				return skipResult{false, "not clearing"}, nil
			}

			ok, err := m.MissingKeyword(c.Clear...)
			if !ok {
				return skipResult{false, fmt.Sprintf("needs to lose labels [%s]", strings.Join(c.Clear, ", "))}, err
			}

			return skipResult{true, fmt.Sprintf("already lost labels [%s]", strings.Join(c.Clear, ", "))}, err
		},

		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsMoving() {
				return skipResult{false, "not moving"}, nil
			}

			fn, err := m.Folder()
			if c.Move != fn {
				return skipResult{false, fmt.Sprintf("not yet in folder [%s]", c.Move)}, err
			}

			return skipResult{true, fmt.Sprintf("already in folder [%s]", c.Move)}, err
		},

		func(m *Message, c *CompiledRule) (skipResult, error) {
			ok, err := m.HasKeyword("\\Starred")
			if ok {
				return skipResult{true, "do not modify \\Starred"}, err
			}

			return skipResult{false, "not \\Starred"}, err
		},
	}

	ruleTests = []ruleTest{
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if !c.HasOkayDate() {
				return testResult{true, "no okay date"}, nil
			}

			(*tests)++

			date, err := m.Date()
			if date.Before(c.OkayDate) {
				return testResult{true, fmt.Sprintf("message is older than okay date [%s]", c.OkayDate.Format(time.RFC3339))}, err
			}

			return testResult{false, fmt.Sprintf("message is newer than okay date [%s]", c.OkayDate.Format(time.RFC3339))}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.From == "" {
				return testResult{true, "no from test"}, nil
			}

			(*tests)++

			from, err := m.AddressList("From")
			return testAddress("From", "from", c.From, from, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.FromDomain == "" {
				return testResult{true, "no from domain test"}, nil
			}

			(*tests)++

			from, err := m.AddressList("From")
			return testDomain("From", "from", c.FromDomain, from, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.To == "" {
				return testResult{true, "no to test"}, nil
			}

			(*tests)++

			to, err := m.AddressList("To")
			return testAddress("To", "to", c.To, to, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.ToDomain == "" {
				return testResult{true, "no to domain test"}, nil
			}

			(*tests)++

			to, err := m.AddressList("To")
			return testDomain("To", "to", c.ToDomain, to, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Sender == "" {
				return testResult{true, "no sender test"}, nil
			}

			(*tests)++

			sender, err := m.AddressList("Sender")
			return testAddress("Sender", "sender", c.Sender, sender, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.DeliveredTo == "" {
				return testResult{true, "no delivered_to test"}, nil
			}

			(*tests)++

			deliveredTo, err := m.AddressList("Delivered-To")
			return testAddress("Delivered-To", "delivered_to", c.DeliveredTo, deliveredTo, err)
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Subject == "" {
				return testResult{true, "no exact subject test"}, nil
			}

			(*tests)++

			subject, err := m.Subject()
			if c.Subject != subject {
				return testResult{false, fmt.Sprintf("message header [Subject] does not exactly match subject test: [%s]", c.Subject)}, err
			}

			return testResult{true, fmt.Sprintf("message header [Subject] exactly matches subject test: [%s]", c.Subject)}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectFold == "" {
				return testResult{true, "no folded case subject test"}, nil
			}

			(*tests)++

			subject, err := m.Subject()
			if !strings.EqualFold(c.SubjectFold, subject) {
				return testResult{false, fmt.Sprintf("message header [Subject] does not match folded case of subject test: [%s]", c.SubjectFold)}, err
			}

			return testResult{true, fmt.Sprintf("message header [Subject] matches folded case of subject test: [%s]", c.SubjectFold)}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectContains == "" {
				return testResult{true, "no subject contains test"}, nil
			}

			(*tests)++

			subject, err := m.Subject()
			if !strings.Contains(subject, c.SubjectContains) {
				return testResult{false, fmt.Sprintf("message header [Subject] fails contains subject test: [%s]", c.SubjectContains)}, err
			}

			return testResult{true, fmt.Sprintf("message header [Subject] passes contains subject test: [%s]", c.SubjectContains)}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectContainsFold == "" {
				return testResult{true, "no subject contains subject folded case test"}, nil
			}

			(*tests)++

			subject, err := m.Subject()
			if !xtrings.ContainsFold(subject, c.SubjectContainsFold) {
				return testResult{false, fmt.Sprintf("message header [Subject] fails contains subject folded case test: [%s]", c.SubjectContainsFold)}, err
			}

			return testResult{true, fmt.Sprintf("message header [Subject] passes cotnains subject folded case test: [%s]", c.SubjectContainsFold)}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Contains == "" {
				return testResult{true, "no contains anywhere test"}, nil
			}

			(*tests)++

			bs, err := m.Raw()
			if !strings.Contains(string(bs), c.Contains) {
				return testResult{false, fmt.Sprintf("message fails contains anywhere test: [%s]", c.Contains)}, err
			}

			return testResult{true, fmt.Sprintf("message passes contains anywhere test: [%s]", c.Contains)}, err
		},

		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.ContainsFold == "" {
				return testResult{true, "no contains anywhere folded case test"}, nil
			}

			(*tests)++

			bs, err := m.Raw()
			if !xtrings.ContainsFold(string(bs), c.ContainsFold) {
				return testResult{false, fmt.Sprintf("message fails contains anywhere folded case test: [%s]", c.ContainsFold)}, err
			}

			return testResult{true, fmt.Sprintf("message passes contains anywhere folded case test: [%s]", c.ContainsFold)}, err
		},
	}
)

func testAddress(dbgh, dbgt, expect string, got addr.AddressList, err error) (testResult, error) {
	if len(got) == 0 {
		return testResult{false, fmt.Sprintf("message is missing [%s] header", dbgh)}, err
	}

	for _, mb := range got.Flatten() {
		if strings.EqualFold(mb.Address(), expect) {
			return testResult{true, fmt.Sprintf("message header [%s] matches [%s] test: [%s]", dbgh, dbgt, expect)}, err
		}
	}

	return testResult{false, fmt.Sprintf("message header [%s] does not match [%s] test: [%s]", dbgh, dbgt, expect)}, err
}

func testDomain(dbgh, dbgt, expect string, got addr.AddressList, err error) (testResult, error) {
	if len(got) == 0 {
		return testResult{false, fmt.Sprintf("message is missing [%s] header", dbgh)}, err
	}

	for _, mb := range got.Flatten() {
		if strings.EqualFold(mb.Domain(), expect) {
			return testResult{true, fmt.Sprintf("message header [%s] matches [%s] domain test: [%s]", dbgh, dbgt, expect)}, err
		}
	}

	return testResult{false, fmt.Sprintf("message header [%s] does not match [%s] domain test: [%s]", dbgh, dbgt, expect)}, err
}

func (m *Message) MoveTo(root string, name string) error {
	if f, ok := labelBoxes[name]; ok {
		name = f
	}

	name = strings.ReplaceAll(name, "/", ".")
	dest := path.Join(root, name)
	if info, err := os.Stat(dest); os.IsExist(err) && info.IsDir() {
		// i.e., I assume this is a mistake.
		return errors.New("folder path does not exist or is not a diretory")
	}

	destFolder := NewMailDirFolder(root, name)
	err := m.r.(*MailDirSlurper).MoveTo(destFolder)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) Save() error {
	mm, err := m.EmailMessage()
	if err != nil {
		return err
	}

	w, err := m.r.(*MailDirSlurper).Replace()
	if err != nil {
		return err
	}
	defer w.Close()

	//fmt.Println("START WRITING")
	_, err = w.Write(mm.Bytes())
	//fmt.Println("END WRITING")
	if err != nil {
		return fmt.Errorf("unable to save %q: %w", m.Filename(), err)
	}

	return nil
}

func (m *Message) BestAlternateFolder() (string, error) {
	ks, err := m.Keywords()
	if err != nil {
		return "", err
	}

	if len(ks[0]) > 0 && strings.Contains(ks[0], "Social") {
		return "JunkSocial", nil
	}

	if len(ks) > 0 {
		return ks[0], nil
	}

	return "gmail.All_Mail", nil
}
