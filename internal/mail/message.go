package mail

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/zostay/go-addr/pkg/addr"
	"github.com/zostay/go-email/v2/message"
	"github.com/zostay/go-email/v2/message/header"

	"github.com/zostay/dotfiles-go/internal/xtrings"
	"github.com/zostay/dotfiles-go/pkg/secrets"
)

var (
	// SASLUser contains the username to use to login
	SASLUser = secrets.MustGet(secrets.Secure, "LABEL_MAIL_USERNAME")

	// SASLPass contsint eh password to use to login
	SASLPass = secrets.MustGet(secrets.Secure, "LABEL_MAIL_PASSWORD")
)

// Message represents a MIME message which may be partially or fully read in.
type Message struct {
	// r is the mechanism used for reading in the message
	r Slurper

	// m is the MIME message representation once parsed
	m message.Generic
}

// NewMessage creates a *Message from a Slurper.
func NewMessage(r Slurper) *Message {
	return &Message{r: r}
}

// NewMailDirMessage returns a single message to be read in using a *DirSlurper from
// the given key, flags, read status, and folder.
func NewMailDirMessage(key, flags, rd string, folder *DirFolder) *Message {
	r := NewMailDirSlurper(key, flags, rd, folder)
	return NewMessage(r)
}

// NewMailDirMessageWithStat returns a single message to be read in using a
// *DirSlurper, but with the given stat info.
func NewMailDirMessageWithStat(key, flags, rd string, folder *DirFolder, fi *os.FileInfo) *Message {
	r := NewMailDirSlurperWithStat(key, flags, rd, folder, fi)
	return NewMessage(r)
}

// NewFileMessage returns a single message to be read in using a *MessageSlurper
// with the given filename.
func NewFileMessage(filename string) *Message {
	r := NewMessageSlurper(filename)
	return NewMessage(r)
}

// Filename returns the name of the file containing the message.
func (m *Message) Filename() string {
	return m.r.Filename()
}

// Stat returns the file info for the file containing the message or an error.
func (m *Message) Stat() (os.FileInfo, error) {
	return m.r.Stat()
}

// EmailMessage returns the message.Opaque with headers parsed or returns an
// error.
func (m *Message) EmailMessage() (message.Generic, error) {
	if m.m != nil {
		return m.m, nil
	}

	r, err := m.r.Reader()
	if err != nil {
		return nil, err
	}

	m.m, err = message.Parse(r, message.WithoutMultipart())
	return m.m, err
}

// MultipartEmailMessage returns the message.Multipart or message.Opaque parsed
// or returns an error.
func (m *Message) MultipartEmailMessage() (message.Generic, error) {
	if m.m != nil {
		return m.m, nil
	}

	r, err := m.r.Reader()
	if err != nil {
		return nil, err
	}

	m.m, err = message.Parse(r,
		message.WithUnlimitedRecursion(),
		message.WithMaxPartLength(message.DefaultMaxPartLength*1_000))
	return m.m, err
}

// Raw returns the byte representation of the original read in message or an
// error.
func (m *Message) Raw() ([]byte, error) {
	r, err := m.r.Reader()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// Date returns the contents of hte Date hread of the message or an error.
func (m *Message) Date() (time.Time, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return time.Time{}, err
	}

	return mm.GetHeader().GetDate()
}

// Keywords returns the contents of the Keywords header of the message as a
// slice of strings or an error.
func (m *Message) Keywords() ([]string, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	ks, err := mm.GetHeader().GetKeywords()
	if errors.Is(err, header.ErrNoSuchField) {
		return []string{}, nil
	}
	return ks, err
}

// KeywordsSet returns the contents of the Keywords header as a set or an error.
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

// HasNonconformingKeywords returns true if the Keywords header is mailformed.
func (m *Message) HasNonconformingKeywords() (bool, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return false, err
	}

	sk, err := mm.GetHeader().GetKeywords()
	if err != nil {
		return false, err
	}

	for _, k := range sk {
		where := strings.IndexFunc(k, func(c rune) bool {
			return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '-' || c == '.' || c == '/'
		})

		if where >= 0 {
			return true, nil
		}
	}

	return false, nil
}

// HasKeyword returns true if the Keywords header contains all of the given
// keyword names. It returns an error if it has a problem reading or parsing the
// Keywords header. If there's no error reading Keywords and the list of names
// is empty, this will always return true.
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

// MissingKeyword returns true if the Keywrods header contains none of the given
// keyword names. It returns an error if it has a problem reading or parsing the
// Keywords header. If there's no error reading Keywords and the list of names
// is empty, this will always return true.
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

// CleanupKeywords removes duplicate keywords from the Keywords header and
// updates the Keywords header. Returns an error if it has a problem reading or
// writing the header.
func (m *Message) CleanupKeywords() error {
	km, err := m.KeywordsSet()
	if err != nil {
		return err
	}

	return m.updateKeywords(km)
}

// AddKeyword adds all the given names to the Keywords header. Returns an error
// if it has a problem reading the email message or writing ot hte Keywords
// header.
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

// updateKeywords is used to update the in-memory representation of the Keywords
// header. Returns ane error if it has a problem reading the email message or
// writing to the Keywords header.
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

	mm.GetHeader().SetKeywords(ks...)
	if err != nil {
		return err
	}

	return nil
}

// RemoveKeyword removes all the a names given from the Keywords header, if
// those keywords are present. Returns an error if it has a problem reading or
// parsing the Keywords header or writing to it.
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

// AllAddressLists tries every header matching the given key and parses it as an
// address list. Those lists are joined together and returned as a single list.
// Returns an error if it has trouble reading the message or parsing the address
// lists.
func (m *Message) AllAddressLists(key string) ([]addr.AddressList, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	return mm.GetHeader().GetAllAddressLists(key)
}

// AddressList tries the first header matching the given key and parses it as an
// address list. It returns the parsed list or returns ane error.
func (m *Message) AddressList(key string) (addr.AddressList, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return nil, err
	}

	return mm.GetHeader().GetAddressList(key)
}

// Subject returns the contents of the Subject header.
func (m *Message) Subject() (string, error) {
	mm, err := m.EmailMessage()
	if err != nil {
		return "", err
	}

	return mm.GetHeader().GetSubject()
}

// Folder returns the name of the folder that contains this email's file.
func (m *Message) Folder() (string, error) {
	return m.r.Folder(), nil
}

// skipTest represents a function used to skip an action when it won't apply.
type skipTest func(*Message, *CompiledRule) (skipResult, error)

// ruleTest represents a function used to determine whether a rule is
// applicable (if it has not been skipped).
type ruleTest func(*Message, *CompiledRule, *int) (testResult, error)

// skipResult describes whether a skip should occur and why
type skipResult struct {
	skip   bool
	reason string
}

// testResult describes whether a rule matches and why
type testResult struct {
	pass   bool
	reason string
}

var (
	// skipTests defines all the ways in which a rule may be skipped
	skipTests = []skipTest{
		// skip because we're labelling and this rule has no label
		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsLabeling() {
				return skipResult{false, cp.Scolor("base", "not labeling")}, nil
			}

			ok, err := m.HasKeyword(c.Label...)
			if !ok {
				return skipResult{false,
					cp.Scolor(
						"base", "needs labels ",
						"label", fmt.Sprintf("%q", strings.Join(c.Label, ", ")),
					),
				}, err
			}

			return skipResult{true,
				cp.Scolor(
					"base", "already labeled ",
					"label", fmt.Sprintf("%q", cp.Join("base", c.Label, ", ")),
				),
			}, err
		},

		// skip because we're clearing and this rule is not a clearing rule
		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsClearing() {
				return skipResult{false, cp.Scolor("base", "not clearing")}, nil
			}

			ok, err := m.MissingKeyword(c.Clear...)
			if !ok {
				return skipResult{false,
					cp.Scolor(
						"base", "needs to lose labels ",
						"label", fmt.Sprintf("%q", strings.Join(c.Clear, ", ")),
					),
				}, err
			}

			return skipResult{true,
				cp.Scolor(
					"base", "already lost labels ",
					"label", fmt.Sprintf("%q", strings.Join(c.Clear, ", ")),
				),
			}, err
		},

		// skip because the message is already in the destination folder
		func(m *Message, c *CompiledRule) (skipResult, error) {
			if !c.IsMoving() {
				return skipResult{false, cp.Scolor("base", "not moving")}, nil
			}

			fn, err := m.Folder()
			if c.Move != fn {
				return skipResult{false,
					cp.Scolor(
						"base", "not yet in folder ",
						"label", fmt.Sprintf("%q", c.Move),
					),
				}, err
			}

			return skipResult{true,
				cp.Scolor(
					"base", "already in folder ",
					"label", fmt.Sprintf("%q", c.Move),
				),
			}, err
		},

		// skip because we do not modify starred messages
		func(m *Message, c *CompiledRule) (skipResult, error) {
			ok, err := m.HasKeyword("\\Starred")
			if ok {
				return skipResult{true,
					cp.Scolor(
						"base", "do not modify ",
						"label", fmt.Sprintf("%q", "\\Starred"),
					),
				}, err
			}

			return skipResult{false,
				cp.Scolor(
					"base", "not ",
					"label", fmt.Sprintf("%q", "\\Starred"),
				),
			}, err
		},
	}

	// ruleTests are the rules that identify which messages match a certain rule
	ruleTests = []ruleTest{
		// match if the message Date is more recent than the ok date
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if !c.HasOkayDate() {
				return testResult{true, cp.Scolor("base", "no okay date")}, nil
			}

			*tests++

			date, err := m.Date()
			if date.Before(c.OkayDate) {
				return testResult{true,
					cp.Scolor(
						"base", "message is older than okay date ",
						"value", fmt.Sprintf("%q", c.OkayDate.Format(time.RFC3339)),
					),
				}, err
			}

			return testResult{false,
				cp.Scolor(
					"base", "message is newer than okay date ",
					"value", fmt.Sprintf("%q", c.OkayDate.Format(time.RFC3339)),
				),
			}, err
		},

		// match if the message has a matching From address
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.From == "" {
				return testResult{true, cp.Scolor("base", "no from test")}, nil
			}

			*tests++

			from, err := m.AddressList("From")
			return testAddress("From", "from", c.From, from, err)
		},

		// match if the message has a matching domain in the From header
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.FromDomain == "" {
				return testResult{true, cp.Scolor("base", "no from domain test")}, nil
			}

			*tests++

			from, err := m.AddressList("From")
			return testDomain("From", "from", c.FromDomain, from, err)
		},

		// match if the message has a matching To address
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.To == "" {
				return testResult{true, cp.Scolor("base", "no to test")}, nil
			}

			*tests++

			to, err := m.AddressList("To")
			return testAddress("To", "to", c.To, to, err)
		},

		// match if the message has a matching domain in the To header
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.ToDomain == "" {
				return testResult{true, cp.Scolor("base", "no to domain test")}, nil
			}

			*tests++

			to, err := m.AddressList("To")
			return testDomain("To", "to", c.ToDomain, to, err)
		},

		// match if the message has a matching Sender address
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Sender == "" {
				return testResult{true, cp.Scolor("base", "no sender test")}, nil
			}

			*tests++

			sender, err := m.AddressList("Sender")
			return testAddress("Sender", "sender", c.Sender, sender, err)
		},

		// match if the message has a matching Delivered-To address
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.DeliveredTo == "" {
				return testResult{true, cp.Scolor("base", "no delivered_to test")}, nil
			}

			*tests++

			deliveredTo, err := m.AllAddressLists("Delivered-To")
			var length int
			for _, dt := range deliveredTo {
				length += len(dt)
			}
			dts := make(addr.AddressList, 0, length)
			for _, dt := range deliveredTo {
				dts = append(dts, dt...)
			}
			return testAddress("Delivered-To", "delivered_to", c.DeliveredTo, dts, err)
		},

		// match if the message has a matching exact Subject header match
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Subject == "" {
				return testResult{true, cp.Scolor("base", "no exact subject test")}, nil
			}

			*tests++

			subject, err := m.Subject()
			if c.Subject != subject {
				return testResult{false,
					cp.Scolor(
						"base", "message header ",
						"header", "\"Subject\"",
						"base", " does not exactly match subject test: ",
						"value", fmt.Sprintf("%q", c.Subject),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", "\"Subject\"",
					"action", " exactly matches subject test: ",
					"value", fmt.Sprintf("%q", c.Subject),
				),
			}, err
		},

		// match if the message has an exact header match, but without case
		// sensitivity
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectFold == "" {
				return testResult{true, cp.Scolor("base", "no folded case subject test")}, nil
			}

			*tests++

			subject, err := m.Subject()
			if !strings.EqualFold(c.SubjectFold, subject) {
				return testResult{false,
					cp.Scolor(
						"base", "message header ",
						"header", "\"Subject\"",
						"base", " does not match folded case of subject test: ",
						"value", fmt.Sprintf("%q", c.SubjectFold),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", "\"Subject\"",
					"action", " matches folded case of subject test: ",
					"value", fmt.Sprintf("%q", c.SubjectFold),
				),
			}, err
		},

		// match if the Subject header contains the given substring
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectContains == "" {
				return testResult{true, cp.Scolor("base", "no subject contains test")}, nil
			}

			*tests++

			subject, err := m.Subject()
			if !strings.Contains(subject, c.SubjectContains) {
				return testResult{false,
					cp.Scolor(
						"base", "message header ",
						"header", "\"Subject\"",
						"base", " fails contains subject test: ",
						"value", fmt.Sprintf("%q", c.SubjectContains),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", "\"Subject\"",
					"action", " passes contains subject test: ",
					"value", fmt.Sprintf("%q", c.SubjectContains),
				),
			}, err
		},

		// match if the Subject header contains the given substring, but using a
		// case-insensitive match
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.SubjectContainsFold == "" {
				return testResult{true, cp.Scolor("base", "no subject contains subject folded case test")}, nil
			}

			*tests++

			subject, err := m.Subject()
			if !xtrings.ContainsFold(subject, c.SubjectContainsFold) {
				return testResult{false,
					cp.Scolor(
						"base", "message header ",
						"header", "\"Subject\"",
						"base", " fails contains subject folded case test: ",
						"value", fmt.Sprintf("%q", c.SubjectContainsFold),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", "\"Subject\"",
					"action", " passes cotnains subject folded case test: ",
					"value", fmt.Sprintf("%q", c.SubjectContainsFold),
				),
			}, err
		},

		// match if the message anywhere contains the given substring
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.Contains == "" {
				return testResult{true, cp.Scolor("base", "no contains anywhere test")}, nil
			}

			*tests++

			bs, err := m.Raw()
			if !strings.Contains(string(bs), c.Contains) {
				return testResult{false,
					cp.Scolor(
						"base", "message fails contains anywhere test: ",
						"value", fmt.Sprintf("%q", c.Contains),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message passes contains anywhere test: ",
					"value", fmt.Sprintf("%q", c.Contains),
				),
			}, err
		},

		// match if the message anywhere contains the given substring, with a
		// case insensitive match
		func(m *Message, c *CompiledRule, tests *int) (testResult, error) {
			if c.ContainsFold == "" {
				return testResult{true, cp.Scolor("base", "no contains anywhere folded case test")}, nil
			}

			*tests++

			bs, err := m.Raw()
			if !xtrings.ContainsFold(string(bs), c.ContainsFold) {
				return testResult{false,
					cp.Scolor(
						"base", "message fails contains anywhere folded case test: ",
						"value", fmt.Sprintf("%q", c.ContainsFold),
					),
				}, err
			}

			return testResult{true,
				cp.Scolor(
					"action", "message passes contains anywhere folded case test: ",
					"value", fmt.Sprintf("%q", c.ContainsFold),
				),
			}, err
		},
	}
)

// testAddress is a function that tests to see if the given addr.AddressList
// contains the expected address. It sets up common diagnostic messages and
// always returns the given err, but formatted with a better diagnostic message.
// The dbgh contains the name of the header for use with diagnostic messages.
// The dbgt contains the name of the test for diagnostic messages.
func testAddress(dbgh, dbgt, expect string, got addr.AddressList, err error) (testResult, error) {
	if err != nil {
		err = fmt.Errorf("error reading %q header: %w", dbgh, err)
	}

	if len(got) == 0 {
		return testResult{false,
			cp.Scolor(
				"base", "message is missing ",
				"header", fmt.Sprintf("%q", dbgh),
				"base", " header",
			),
		}, err
	}

	for _, mb := range got.Flatten() {
		if strings.EqualFold(mb.Address(), expect) {
			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", fmt.Sprintf("%q", dbgh),
					"action", fmt.Sprintf(" matches %q test: ", dbgt),
					"value", fmt.Sprintf("%q", expect),
				),
			}, err
		}
	}

	return testResult{false,
		cp.Scolor(
			"base", "message header ",
			"header", fmt.Sprintf("%q", dbgh),
			"base", fmt.Sprintf(" does not match %q test: ", dbgt),
			"value", fmt.Sprintf("%q", expect),
		),
	}, err
}

// testDomain is a helper that tests to see if the given domain is found in the
// addr.AddressList. It adds diagnostics around the process. The dbgh names the
// header being tested. The dbgt is the test being performed. And the err is
// returned.
func testDomain(dbgh, dbgt, expect string, got addr.AddressList, err error) (testResult, error) {
	if len(got) == 0 {
		return testResult{false,
			cp.Scolor(
				"base", "message is missing ",
				"header", fmt.Sprintf("%q", dbgh),
				"base", " header",
			),
		}, err
	}

	for _, mb := range got.Flatten() {
		if strings.EqualFold(mb.Domain(), expect) {
			return testResult{true,
				cp.Scolor(
					"action", "message header ",
					"header", fmt.Sprintf("%q", dbgh),
					"action", fmt.Sprintf(" matches %q domain test: ", dbgt),
					"value", fmt.Sprintf("%q", expect),
				),
			}, err
		}
	}

	return testResult{false,
		cp.Scolor(
			"base", "message header",
			"header", fmt.Sprintf("%q", dbgh),
			"base", fmt.Sprintf(" does not match %q domain test: ", dbgt),
			"value", fmt.Sprintf("%q", expect),
		),
	}, err
}

// MoveTo moves the message to the maildir folder represented by the given root
// directory and folder name. Returns an error if the move fails.
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
	err := m.r.(*DirSlurper).MoveTo(destFolder)
	if err != nil {
		return err
	}

	return nil
}

// Save saves any modifications made to the message to disk.
func (m *Message) Save() error {
	mm, err := m.EmailMessage()
	if err != nil {
		return err
	}

	w, err := m.r.(*DirSlurper).Replace()
	if err != nil {
		return err
	}
	defer w.Close()

	// fmt.Println("START WRITING")
	_, err = mm.WriteTo(w)
	// fmt.Println("END WRITING")
	if err != nil {
		return fmt.Errorf("unable to save %q: %w", m.Filename(), err)
	}

	return nil
}

// BestAlternateFolder returns a folder name describing a better folder for the
// given name. This really ought to be in configuration.
func (m *Message) BestAlternateFolder() (string, error) {
	ks, err := m.Keywords()
	if err != nil {
		return "", err
	}

	if len(ks) > 0 && strings.Contains(ks[0], "Social") {
		return "JunkSocial", nil
	}

	if len(ks) > 0 {
		return ks[0], nil
	}

	return "gmail.All_Mail", nil
}
