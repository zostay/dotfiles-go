package mail

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-maildir"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/zostay/dotfiles-go/internal/dotfiles"
	"github.com/zostay/dotfiles-go/internal/xtrings"
)

var (
	FromEmail = dotfiles.MustGetSecret("GIT_EMAIL_HOME")
	SASLUser  = dotfiles.MustGetSecret("LABEL_MAIL_USERNAME")
	SASLPass  = dotfiles.MustGetSecret("LABEL_MAIL_PASSWORD")
)

var (
	brokenStarts map[byte][]brokenFix

	fixes = []brokenFix{
		{[]byte("Content-Transfer-Encoding: 8-bit"),
			[]byte("Content-Transfer-Encoding: 8bit")},
	}
)

type brokenFix struct {
	broken []byte
	fix    []byte
}

func init() {
	brokenStarts = make(map[byte][]brokenFix)
	for _, bf := range fixes {
		c := bf.broken[0]
		if ks, ok := brokenStarts[c]; ok {
			brokenStarts[c] = append(ks, bf)
		} else {
			brokenStarts[c] = []brokenFix{bf}
		}
	}
}

type Message struct {
	key    string
	folder maildir.Dir
	e      *message.Entity
	m      *mail.Reader
}

func NewMessage(folder maildir.Dir, key string) *Message {
	return &Message{
		key:    key,
		folder: folder,
	}
}

func byteEqual(b1 []byte, s, e int, b2 []byte) bool {
	for i := 0; i < e-s; i++ {
		if b1[i+s] != b2[i] {
			return false
		}
	}
	return true
}

func fixHeadersReader(r io.Reader) (*bytes.Reader, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	os := make([]byte, 0, len(bs))

ByteLoop:
	for i, b := range bs {
		bfs, ok := brokenStarts[b]
		if ok && (i == 0 || bs[i-1] == '\r' || bs[i-1] == '\n') {
			for _, bf := range bfs {
				if len(bs)-i < len(bf.broken) && byteEqual(bs, i, i+len(bf.broken), bf.broken) {
					os = append(os, bf.fix...)
					continue ByteLoop
				}
			}
		} else {
			os = append(os, b)
		}
	}

	return bytes.NewReader(os), err
}

func (m *Message) Message() (*mail.Reader, error) {
	if m.m != nil {
		return m.m, nil
	}

	r, err := m.folder.Open(m.key)
	if err != nil {
		return nil, err
	}

	defer r.Close()

	fr, err := fixHeadersReader(r)
	if err != nil {
		return nil, err
	}

	m.e, err = message.Read(fr)
	m.m = mail.NewReader(m.e)
	if err != nil {
		return m.m, fmt.Errorf("unable to parse email entity for mail %s/*/%s: %w", m.folder, m.key, err)
	} else {
		return m.m, nil
	}
}

func (m *Message) Raw() ([]byte, error) {
	r, err := m.folder.Open(m.key)
	if err != nil {
		return []byte{}, err
	}

	defer r.Close()

	return ioutil.ReadAll(r)
}

func (m *Message) Reader() (io.ReadCloser, error) {
	return m.folder.Open(m.key)
}

func (m *Message) Date() (time.Time, error) {
	msg, err := m.Message()
	if err != nil {
		return time.Time{}, err
	}

	return msg.Header.Date()
}

func (m *Message) Keywords() ([]string, error) {
	msg, err := m.Message()
	if err != nil {
		return []string{}, err
	}

	sk := msg.Header.Get("Keywords")
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
	msg, err := m.Message()
	if err != nil {
		return false, err
	}

	sk := msg.Header.Get("Keywords")
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

	m.m.Header.Set("Keywords", strings.Join(ks, " "))

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
	msg, err := m.Message()
	if err != nil {
		return err
	}

	ks := make([]string, 0, len(km))
	for k := range km {
		ks = append(ks, k)
	}

	sort.Strings(ks)

	msg.Header.Set("Keywords", strings.Join(ks, ", "))

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

func (m *Message) AddressList(key string) ([]*mail.Address, error) {
	var addr []*mail.Address

	msg, err := m.Message()
	if err != nil {
		return addr, err
	}

	addr, err = msg.Header.AddressList(key)
	if err != nil {
		return addr, fmt.Errorf("unable to read address list of header %s: %w", key, err)
	}

	return addr, nil
}

func (m *Message) Subject() (string, error) {
	msg, err := m.Message()
	if err != nil {
		return "", err
	}

	return msg.Header.Subject()
}

func (m *Message) Folder() (string, error) {
	return path.Base(string(m.folder)), nil
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
			if date.After(c.OkayDate) {
				return testResult{true, fmt.Sprintf("message is more recent than okay date [%s]", c.OkayDate.Format(time.RFC3339))}, err
			}

			return testResult{false, fmt.Sprintf("message is older than okay date [%s]", c.OkayDate.Format(time.RFC3339))}, err
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

func testAddress(dbgh, dbgt, expect string, got []*mail.Address, err error) (testResult, error) {
	if len(got) == 0 {
		return testResult{false, fmt.Sprintf("message is missing [%s] header", dbgh)}, err
	}

	for _, addr := range got {
		if strings.EqualFold(addr.Address, expect) {
			return testResult{true, fmt.Sprintf("message header [%s] matches [%s] test: [%s]", dbgh, dbgt, expect)}, err
		}
	}

	return testResult{false, fmt.Sprintf("message header [%s] does not match [%s] test: [%s]", dbgh, dbgt, expect)}, err
}

func testDomain(dbgh, dbgt, expect string, got []*mail.Address, err error) (testResult, error) {
	if len(got) == 0 {
		return testResult{false, fmt.Sprintf("message is missing [%s] header", dbgh)}, err
	}

	for _, addr := range got {
		idx := strings.IndexRune(addr.Address, '@')
		d := addr.Address[idx:]
		if strings.EqualFold(d, expect) {
			return testResult{true, fmt.Sprintf("message header [%s] matches [%s] domain test: [%s]", dbgh, dbgt, expect)}, err
		}
	}

	return testResult{false, fmt.Sprintf("message header [%s] does not match [%s] domain test: [%s]", dbgh, dbgt, expect)}, err
}

func (m *Message) ForwardTo(tos ...string) error {
	auth := sasl.NewPlainClient("", SASLUser, SASLPass)

	msg, err := m.Message()
	if err != nil {
		return err
	}

	zfw := msg.Header.Get("X-Zostay-Forwarded")
	zfwm := make(map[string]struct{})
	zfws := make([]string, 0, len(tos))
	if zfw != "" {
		zfws = strings.FieldsFunc(zfw, func(c rune) bool {
			return unicode.IsSpace(c) || c == ','
		})
		for _, e := range zfws {
			zfwm[e] = struct{}{}
		}
	}

	finalTos := make([]string, 0, len(tos))
	for _, to := range tos {
		if _, ok := zfwm[to]; !ok {
			finalTos = append(finalTos, to)
			zfws = append(zfws, to)
		}
	}

	r, err := m.Reader()
	if err != nil {
		return err
	}

	err = smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		FromEmail,
		finalTos,
		r,
	)
	if err != nil {
		return err
	}

	sort.Strings(zfws)

	msg.Header.Set("X-Zostay-Forwarded", strings.Join(zfws, ", "))

	return nil
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

	destFolder := maildir.Dir(dest)
	err := m.folder.Move(destFolder, m.key)
	if err != nil {
		return err
	}

	m.folder = destFolder

	return nil
}

func (m *Message) Save() error {
	msg, err := m.Message()
	if err != nil {
		return err
	}

	m.e.Header = msg.Header.Header
	key, w, err := m.folder.Create([]maildir.Flag{})
	if err != nil {
		return err
	}

	err = m.folder.Remove(m.key)
	if err != nil {
		return err
	}

	m.key = key
	err = m.e.WriteTo(w)
	if err != nil {
		return err
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
