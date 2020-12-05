package mail

import (
	"bytes"
	"errors"
	"fmt"
	"html"
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

const (
	FromName = "Andrew Sterling Hanenkamp"

	ForwardedMessagePrefix = "---------- Forwarded message ---------"
)

var (
	FromEmail = dotfiles.MustGetSecret("GIT_EMAIL_HOME")

	SASLUser = dotfiles.MustGetSecret("LABEL_MAIL_USERNAME")
	SASLPass = dotfiles.MustGetSecret("LABEL_MAIL_PASSWORD")
)

var (
	brokenStarts map[byte][]brokenFix

	fixes = []brokenFix{
		{[]byte("Content-Transfer-Encoding: 8-bit"),
			[]byte("Content-Transfer-Encoding: 8bit")},
	}

	FromEmailAddress AddressList
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

	FromEmailAddress = make(AddressList, 1)
	FromEmailAddress[0] = &Address{
		Name:    FromName,
		Address: FromEmail,
	}
}

type Message struct {
	r Opener
	h *mail.Header
}

func NewMessage(r Opener) *Message {
	return &Message{r: r}
}

func NewMailDirMessage(folder maildir.Dir, key string) *Message {
	r := NewMailDirOpener(folder, key)
	return NewMessage(r)
}

func NewFileMessage(filename string) *Message {
	r := NewMessageOpener(filename)
	return NewMessage(r)
}

func byteEqual(b1 []byte, b2 []byte) bool {
	return strings.EqualFold(string(b1), string(b2))
}

func fixHeadersReader(r io.Reader) (io.Reader, error) {
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var os bytes.Buffer
	var lb byte

ByteLoop:
	for i := 0; len(bs) > 0; i++ {
		b := bs[0]
		bfs, ok := brokenStarts[b]
		if ok && (i == 0 || lb == '\r' || lb == '\n') {
			for _, bf := range bfs {
				if len(bs) >= len(bf.broken) && byteEqual(bs[0:len(bf.broken)], bf.broken) {
					os.Write(bf.fix)
					lb = bs[len(bf.broken)-1]
					bs = bs[len(bf.broken):]
					continue ByteLoop
				}
			}
			lb = b
			bs = bs[1:]
			os.WriteByte(b)
		} else {
			lb = b
			bs = bs[1:]
			os.WriteByte(b)
		}
	}

	return &os, err
}

func (m *Message) EmailEntity() (*message.Entity, error) {
	r, err := m.r.Open()
	if err != nil {
		return nil, err
	}

	defer r.Close()

	fr, err := fixHeadersReader(r)
	if err != nil {
		return nil, err
	}

	e, err := message.Read(fr)
	if err != nil {
		f, _ := m.r.Filename()
		return nil, fmt.Errorf("unable to parse email entity for mail %s: %w", f, err)
	} else {
		return e, nil
	}
}

func (m *Message) EmailReader() (*mail.Reader, error) {
	e, err := m.EmailEntity()
	if err != nil {
		return nil, err
	} else {
		m := mail.NewReader(e)
		return m, nil
	}
}

func (m *Message) EmailHeader() (*mail.Header, error) {
	if m.h != nil {
		return m.h, nil
	}

	msg, err := m.EmailReader()
	if err != nil {
		return nil, err
	} else {
		m.h = &msg.Header
		return &msg.Header, nil
	}
}

func (m *Message) Raw() ([]byte, error) {
	r, err := m.r.Open()
	if err != nil {
		return []byte{}, err
	}

	defer r.Close()

	return ioutil.ReadAll(r)
}

func (m *Message) Reader() (io.ReadCloser, error) {
	return m.r.Open()
}

func (m *Message) ForwardReader(to AddressList) (*bytes.Buffer, error) {
	r, err := m.EmailReader()
	if err != nil {
		return nil, err
	}

	var (
		h   mail.Header
		buf bytes.Buffer
	)

	h.SetDate(time.Now())
	h.SetAddressList("To", to)
	h.SetAddressList("From", FromEmailAddress)
	h.SetAddressList("X-Forwarded-To", to)
	h.SetAddressList("X-Forwarded-For", FromEmailAddress)

	fwdSubject, err := r.Header.Subject()
	if err != nil {
		return nil, err
	}

	h.SetSubject("Fwd: " + fwdSubject)

	fwdFromList, err := h.AddressList("From")
	if err != nil {
		return nil, err
	}

	fwdToList, err := h.AddressList("To")
	if err != nil {
		return nil, err
	}

	fwdCcList, err := h.AddressList("Cc")
	if err != nil {
		return nil, err
	}

	fwdDate, err := h.Date()
	if err != nil {
		return nil, err
	}

	w, err := mail.CreateWriter(&buf, h)
	if err != nil {
		return nil, err
	}

	ip, err := w.CreateInline()
	if err != nil {
		return nil, err
	}

	for {
		pr, err := r.NextPart()
		if err == io.EOF {
			break
		}

		switch ph := pr.Header.(type) {
		case *mail.InlineHeader:
			ct, ps, err := ph.ContentType()
			if err != nil {
				return nil, err
			}

			pfh := mail.InlineHeader{}
			pfh.SetContentType(ct, ps)

			pw, err := ip.CreatePart(pfh)
			if err != nil {
				return nil, err
			}

			if ct == "text/plain" {
				_, _ = io.WriteString(pw, ForwardedMessagePrefix)
				_, _ = io.WriteString(pw, "\nFrom: "+AddressListString(fwdFromList))
				_, _ = io.WriteString(pw, "\nDate: "+fwdDate.Format(time.RFC1123))
				_, _ = io.WriteString(pw, "\nSubject: "+fwdSubject)
				_, _ = io.WriteString(pw, "\nTo: "+AddressListString(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = io.WriteString(pw, "\nCc: "+AddressListString(fwdCcList))
				}
				_, _ = io.WriteString(pw, "\n\n")
			} else if ct == "text/html" {
				_, _ = io.WriteString(pw, "<div><br></div><div><br><div>")
				_, _ = io.WriteString(pw, ForwardedMessagePrefix)
				_, _ = io.WriteString(pw, "<br>From: "+AddressListHTML(fwdFromList))
				_, _ = io.WriteString(pw, "<br>Date: "+fwdDate.Format(time.RFC1123))
				_, _ = io.WriteString(pw, "<br>Subject: "+html.EscapeString(fwdSubject))
				_, _ = io.WriteString(pw, "<br>To: "+AddressListHTML(fwdToList))
				if len(fwdCcList) > 0 {
					_, _ = io.WriteString(pw, "<br>Cc: "+AddressListHTML(fwdCcList))
				}
				_, _ = io.WriteString(pw, "<br></div><br><br>")
			}

			_, _ = io.Copy(pw, pr.Body)
			pw.Close()

		case *mail.AttachmentHeader:
			ct, ps, err := ph.ContentType()
			if err != nil {
				return nil, err
			}

			pfh := mail.AttachmentHeader{}
			pfh.SetContentType(ct, ps)

			pw, err := w.CreateAttachment(pfh)
			if err != nil {
				return nil, err
			}

			_, _ = io.Copy(pw, pr.Body)
			pw.Close()
		}
	}

	if ip != nil {
		ip.Close()
	}

	w.Close()

	return &buf, nil
}

func (m *Message) Date() (time.Time, error) {
	h, err := m.EmailHeader()
	if err != nil {
		return time.Time{}, err
	}

	return h.Date()
}

func (m *Message) Keywords() ([]string, error) {
	h, err := m.EmailHeader()
	if err != nil {
		return []string{}, err
	}

	sk := h.Get("Keywords")
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
	h, err := m.EmailHeader()
	if err != nil {
		return false, err
	}

	sk := h.Get("Keywords")
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

	h, err := m.EmailHeader()
	if err != nil {
		return err
	}

	h.Set("Keywords", strings.Join(ks, " "))

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
	h, err := m.EmailHeader()
	if err != nil {
		return err
	}

	ks := make([]string, 0, len(km))
	for k := range km {
		ks = append(ks, k)
	}

	sort.Strings(ks)

	h.Set("Keywords", strings.Join(ks, ", "))

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

func (m *Message) AddressList(key string) (AddressList, error) {
	var addr []*mail.Address

	h, err := m.EmailHeader()
	if err != nil {
		return addr, err
	}

	addr, err = h.AddressList(key)
	if err != nil {
		return addr, fmt.Errorf("unable to read address list of header %s: %w", key, err)
	}

	return addr, nil
}

func (m *Message) Subject() (string, error) {
	h, err := m.EmailHeader()
	if err != nil {
		return "", err
	}

	return h.Subject()
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

func (m *Message) ForwardTo(tos ...*Address) error {
	auth := sasl.NewPlainClient("", SASLUser, SASLPass)

	h, err := m.EmailHeader()
	if err != nil {
		return err
	}

	zfw := h.Get("X-Zostay-Forwarded")
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
		if _, ok := zfwm[to.Address]; !ok {
			finalTos = append(finalTos, to.Address)
			zfws = append(zfws, to.Address)
		}
	}

	r, err := m.ForwardReader(tos)
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

	m.h.Set("X-Zostay-Forwarded", strings.Join(zfws, ", "))

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
	err := m.r.(*MailDirOpener).MoveTo(destFolder)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) Save() error {
	h, err := m.EmailHeader()
	if err != nil {
		return err
	}

	e, err := m.EmailEntity()
	if err != nil {
		return err
	}

	w, err := m.r.(*MailDirOpener).Replace()
	if err != nil {
		return err
	}

	//fmt.Println("START WRITING")
	e.Header = h.Header
	err = e.WriteTo(w)
	//fmt.Println("END WRITING")
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
