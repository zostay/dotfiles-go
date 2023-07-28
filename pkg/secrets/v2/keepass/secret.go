package keepass

import (
	"fmt"
	"net/url"
	"time"

	keepass "github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
	"github.com/zostay/go-std/set"

	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

const (
	keyTitle    = "Title"
	keyUsername = "Username"
	keySecret   = "Password"
	keyType     = "Type"
	keyURL      = "URL"
)

var stdKeys = set.New(keyTitle, keyUsername, keySecret, keyType, keyURL)

type Secret struct {
	db  *keepass.Database
	e   *keepass.Entry
	dir string

	newFields   map[string]string
	delFields   set.Set[string]
	newUrl      *url.URL
	newLocation *string
}

func newSecret(
	db *keepass.Database,
	e *keepass.Entry,
	dir string,
) *Secret {
	return &Secret{
		db:        db,
		e:         e,
		dir:       dir,
		newFields: map[string]string{},
		delFields: set.New[string](),
	}
}

func fromSecret(
	db *keepass.Database,
	secret secrets.Secret,
	keepID bool,
) *Secret {
	if eSec, isESec := secret.(*Secret); isESec {
		cp := eSec.e.Clone()
		retSec := &Secret{
			db:        db,
			e:         &cp,
			dir:       eSec.dir,
			newFields: map[string]string{},
			delFields: set.New[string](),
		}

		if !keepID {
			var zero keepass.UUID
			copy(retSec.e.UUID[:], zero[:])
		}

		retSec.applyChanges(secret)
		return retSec
	}

	var uuid keepass.UUID
	if keepID {
		uuid, _ = makeUUID(secret.ID())
	}

	eSec := &Secret{
		db: db,
		e: &keepass.Entry{
			UUID:   uuid,
			Values: make([]keepass.ValueData, 0, len(secret.Fields())+stdKeys.Len()),
		},
		dir: secret.Location(),
	}

	eSec.applyChanges(secret)
	return eSec
}

// setEntryValue replaces a value in an entry or adds the value to the entry
func (s *Secret) setEntryValue(key, value string, protected bool) {
	// update existing
	for k, v := range s.e.Values {
		if v.Key == key {
			s.e.Values[k].Value.Content = value
			return
		}
	}

	// create new
	newValue := keepass.ValueData{
		Key: key,
		Value: keepass.V{
			Content:   value,
			Protected: w.NewBoolWrapper(protected),
		},
	}
	s.e.Values = append(s.e.Values, newValue)
}

func (s *Secret) applyChanges(secret secrets.Secret) {
	for k, v := range secret.Fields() {
		s.setEntryValue(k, v, false)
	}

	s.setEntryValue(keyTitle, secret.Name(), false)
	s.setEntryValue(keyUsername, secret.Username(), false)
	s.setEntryValue(keySecret, secret.Password(), true)
	s.setEntryValue(keyType, secret.Type(), false)
	s.setEntryValue(keyURL, secret.Url().String(), false)
}

func makeID(id keepass.UUID) string {
	t, _ := id.MarshalText()
	return string(t)
}

func makeUUID(id string) (keepass.UUID, error) {
	var uuid keepass.UUID
	err := uuid.UnmarshalText([]byte(id))
	return uuid, err
}

func (s *Secret) Len() int {
	return len(s.e.Values) + len(s.newFields)
}

func (s *Secret) set(key, value string) {
	s.newFields[key] = value
	s.delFields.Delete(key)
}

func (s *Secret) ID() string {
	return makeID(s.e.UUID)
}

func (s *Secret) Name() string {
	if title, hasNewTitle := s.newFields[keyTitle]; hasNewTitle {
		return title
	}
	return s.e.GetTitle()
}

func (s *Secret) SetName(name string) {
	s.set(keyTitle, name)
}

func (s *Secret) Username() string {
	if username, hasNewUsername := s.newFields[keyUsername]; hasNewUsername {
		return username
	}
	return s.e.GetContent(keyUsername)
}

func (s *Secret) SetUsername(username string) {
	s.set(keyUsername, username)
}

func (s *Secret) whileUnlocked(run func()) {
	err := s.db.UnlockProtectedEntries()
	if err != nil {
		panic(fmt.Errorf("failed to unlock protected entries: %w", err))
	}
	defer func() {
		err := s.db.LockProtectedEntries()
		if err != nil {
			panic(fmt.Errorf("failed to lock protected entries: %w", err))
		}
	}()
	run()
}

func (s *Secret) Password() string {
	if secret, hasNewSecret := s.newFields[keySecret]; hasNewSecret {
		return secret
	}

	var secret string
	s.whileUnlocked(func() {
		secret = s.e.GetPassword()
	})
	return secret
}

func (s *Secret) SetPassword(secret string) {
	s.set(keySecret, secret)
}

func (s *Secret) Type() string {
	if typ, hasNewType := s.newFields[keyType]; hasNewType {
		return typ
	}
	return s.e.GetContent(keyType)
}

func (s *Secret) SetType(typ string) {
	s.set(keyType, typ)
}

func (s *Secret) Fields() map[string]string {
	flds := make(map[string]string, len(s.e.Values))
	for _, val := range s.e.Values {
		if stdKeys.Contains(val.Key) {
			continue
		}
		if newValue, hasNewValue := s.newFields[val.Key]; hasNewValue {
			flds[val.Key] = newValue
			continue
		}
		flds[val.Key] = val.Value.Content
	}
	return flds
}

func (s *Secret) GetField(key string) string {
	if stdKeys.Contains(key) {
		return ""
	}

	if newValue, hasNewValue := s.newFields[key]; hasNewValue {
		return newValue
	}
	return s.e.GetContent(key)
}

func (s *Secret) SetField(key, value string) {
	if key == keySecret {
		s.SetPassword(value)
	}
	s.set(key, value)
}

func (s *Secret) DeleteField(key string) {
	s.delFields.Insert(key)
}

func (s *Secret) LastModified() time.Time {
	return s.e.Times.LastModificationTime.Time
}

func (s *Secret) Url() *url.URL {
	if s.newUrl != nil {
		return s.newUrl
	}
	urlStr := s.e.GetContent(keyURL)
	u, _ := url.Parse(urlStr)
	return u
}

func (s *Secret) SetUrl(u *url.URL) {
	s.newUrl = u
}

func (s *Secret) Location() string {
	if s.newLocation != nil {
		return *s.newLocation
	}
	return s.dir
}

func (s *Secret) SetLocation(loc string) {
	s.newLocation = &loc
}
