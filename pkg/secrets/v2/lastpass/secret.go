package lastpass

import (
	"net/url"
	"strconv"
	"time"

	"github.com/ansd/lastpass-go"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

type Secret struct {
	*lastpass.Account

	parsed bool
	typ    string
	notes  map[string]string
}

func newSecret(account *lastpass.Account) *Secret {
	return &Secret{
		Account: account,
	}
}

func fromSecret(secret secrets.Secret) *Secret {
	newSec := newSecret(
		&lastpass.Account{
			ID:       secret.ID(),
			Name:     secret.Name(),
			Username: secret.Username(),
			Password: secret.Password(),
			URL:      secret.Url().String(),
			Group:    secret.Location(),
			Notes:    writeNotes(secret.Type(), secret.Fields()),
		},
	)

	if s, isSecret := secret.(*Secret); isSecret && s.parsed {
		newSec.Notes = writeNotes(s.typ, s.notes)
	}

	return newSec
}

func (s *Secret) ID() string {
	return s.Account.ID
}

func (s *Secret) Name() string {
	return s.Account.Name
}

func (s *Secret) SetName(name string) {
	s.Account.Name = name
}

func (s *Secret) Username() string {
	return s.Account.Username
}

func (s *Secret) SetUsername(username string) {
	s.Account.Username = username
}

func (s *Secret) Password() string {
	return s.Account.Password
}

func (s *Secret) SetPassword(secret string) {
	s.Account.Password = secret
}

func (s *Secret) Url() *url.URL {
	url, _ := url.Parse(s.Account.URL)
	return url
}

func (s *Secret) SetUrl(url *url.URL) {
	s.Account.URL = url.String()
}

func (s *Secret) Location() string {
	return s.Account.Group
}

func (s *Secret) parseNotes() {
	if s.parsed {
		return
	}

	flds := parseNotes(s.Account.Notes)
	s.typ = flds["NoteType"]
	delete(flds, "NoteType")
	s.notes = flds
	s.parsed = true
}

func (s *Secret) Type() string {
	s.parseNotes()
	return s.typ
}

func (s *Secret) SetType(typ string) {
	s.parseNotes()
	s.typ = typ
}

func (s *Secret) Fields() map[string]string {
	s.parseNotes()
	return s.notes
}

func (s *Secret) GetField(name string) string {
	s.parseNotes()
	return s.notes[name]
}

func (s *Secret) SetFields(fields map[string]string) {
	s.parseNotes()
	s.notes = fields
}

func (s *Secret) SetField(name, value string) {
	s.parseNotes()
	s.notes[name] = value
}

func (s *Secret) makeNotes() string {
	if s.parsed {
		return writeNotes(s.typ, s.notes)
	}
	return s.Account.Notes
}

func (s *Secret) LastModified() time.Time {
	lmSeconds, _ := strconv.ParseInt(s.Account.LastModifiedGMT, 10, 64)
	return time.Unix(lmSeconds, 0)
}
