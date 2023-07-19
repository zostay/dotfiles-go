package secrets

import (
	"net/url"
	"time"
)

// SingleOption is a function that can be used to modify a secret.
type SingleOption func(*Single)

// WithType sets the type of the secret.
func WithType(typ string) SingleOption {
	return func(s *Single) {
		s.typ = typ
	}
}

// WithLastModified sets the last modified time for the secret.
func WithLastModified(t time.Time) SingleOption {
	return func(s *Single) {
		s.lastModified = t
	}
}

// WithUrl sets the URL for the secret.
func WithUrl(u *url.URL) SingleOption {
	return func(s *Single) {
		s.url = u
	}
}

// WithLocation sets the location for the secret.
func WithLocation(l string) SingleOption {
	return func(s *Single) {
		s.location = l
	}
}

// WithField sets a field on the secret.
func WithField(name, value string) SingleOption {
	return func(s *Single) {
		s.fields[name] = value
	}
}

// WithFields sets the given fields on the secret.
func WithFields(fields map[string]string) SingleOption {
	return func(s *Single) {
		for name, value := range fields {
			s.fields[name] = value
		}
	}
}

// Single represents a single secret stored in a Keeper.
type Single struct {
	id       string // the unique identifier
	name     string // the name given to the secret
	username string // the username to store
	secret   string // the secret/password/key associated with the secret

	typ    string            // the type of the secret
	fields map[string]string // additional fields associated with the secret

	lastModified time.Time // the time the secret was last modified
	url          *url.URL  // the URL associated with the secret
	location     string    // the location/group the secret is in
}

// NewSecret creates a secret from the given settings.
func NewSecret(id, name, username, secret string, opts ...SingleOption) *Single {
	sec := &Single{
		id:       id,
		name:     name,
		username: username,
		secret:   secret,
		fields:   map[string]string{},
	}

	for _, opt := range opts {
		opt(sec)
	}

	return sec
}

// NewSecretFromSecret creates a *Single from the given secret.
func NewSingleFromSecret(s Secret) *Single {
	return &Single{
		id:       s.ID(),
		name:     s.Name(),
		username: s.Username(),
		secret:   s.Secret(),

		typ:    s.Type(),
		fields: s.Fields(),

		lastModified: s.LastModified(),
		url:          s.Url(),
		location:     s.Location(),
	}
}

// ID returns the unique identifier for the secret.
func (s *Single) ID() string {
	return s.id
}

// Name returns the name of the secret.
func (s *Single) Name() string {
	return s.name
}

// SetName sets the name of the secret.
func (s *Single) SetName(name string) {
	s.name = name
}

// Username returns the username of the secret.
func (s *Single) Username() string {
	return s.username
}

// SetUsername sets the username of the secret.
func (s *Single) SetUsername(username string) {
	s.username = username
}

// Single returns the secret of the secret.
func (s *Single) Secret() string {
	return s.secret
}

// SetSecret sets the secret of the secret.
func (s *Single) SetSecret(secret string) {
	s.secret = secret
}

// Type returns the type of the secret.
func (s *Single) Type() string {
	return s.typ
}

// SetType sets the type of the secret.
func (s *Single) SetType(typ string) {
	s.typ = typ
}

// Fields returns the fields of the secret.
func (s *Single) Fields() map[string]string {
	return s.fields
}

// LastModified returns the last modified time of the secret.
func (s *Single) LastModified() time.Time {
	return s.lastModified
}

// SetLastModified sets the last modified time of the secret.
func (s *Single) SetLastModified(lastModified time.Time) {
	s.lastModified = lastModified
}

// Url returns the URL of the secret.
func (s *Single) Url() *url.URL {
	return s.url
}

// SetUrl sets the URL of the secret.
func (s *Single) SetUrl(url *url.URL) {
	s.url = url
}

// Location returns the location of the secret.
func (s *Single) Location() string {
	return s.location
}

// SetLocation sets the location of the secret.
func (s *Single) SetLocation(location string) {
	s.location = location
}

// GetField returns the value of the named field. This works safely whether
// Field has been initialized or not.
func (s *Single) GetField(name string) string {
	if s.fields == nil {
		return ""
	}
	return s.fields[name]
}

// SetField sets the value of the named field. This works safely whether Field
// is initialized or not.
func (s *Single) SetField(name, value string) {
	if s.fields == nil {
		s.fields = map[string]string{}
	}
	s.fields[name] = value
}

// DeleteField sets the value of the named field. This works safely whether
// Field is initialized or not.
func (s *Single) DeleteField(name string) {
	if s.fields == nil {
		return
	}
	delete(s.fields, name)
}
