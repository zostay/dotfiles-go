package secrets

import (
	"net/url"
	"time"
)

type Secret interface {
	// ID returns the unique ID of the secret.
	ID() string

	// Name returns the name of the secret.
	Name() string

	// Username returns the username for the secret.
	Username() string

	// Secret returns the secret value.
	Password() string

	// Type returns the type of the secret.
	Type() string

	// Fields returns the fields for the secret.
	Fields() map[string]string

	// GetField returns the value of the named field.
	GetField(string) string

	// LastModified returns the last modified time for the secret.
	LastModified() time.Time

	// Url returns the URL for the secret.
	Url() *url.URL

	// Location returns the location for the secret.
	Location() string
}

type SettableName interface {
	// SetName sets the name of the secret.
	SetName(string)
}

type SettableUsername interface {
	// SetUsername sets the username for the secret.
	SetUsername(string)
}

type SettablePassword interface {
	// SetSecret sets the secret value.
	SetPassword(string)
}

type SettableType interface {
	// SetType sets the type of the secret.
	SetType(string)
}

type SettableFields interface {
	SetField(string, string)
	DeleteField(string)
}

type SettableLastModified interface {
	// SetLastModified sets the last modified time for the secret.
	SetLastModified(time.Time)
}

type SettableUrl interface {
	// SetUrl sets the URL for the secret.
	SetUrl(*url.URL)
}
