package secrets

import (
	"net/url"
	"time"

	"github.com/zostay/go-std/maps"
	"github.com/zostay/go-std/set"
)

type modifier struct {
	base Secret

	name         *string
	username     *string
	secret       *string
	typ          *string
	fields       map[string]string
	removeFields set.Set[string]
	lastModified *time.Time
	url          *url.URL
	location     *string
}

func (m *modifier) ID() string {
	return m.base.ID()
}

func (m *modifier) Name() string {
	if m.name != nil {
		return *m.name
	}
	return m.base.Name()
}

func (m *modifier) SetName(name string) {
	if m.base.(SettableName) != nil {
		m.base.(SettableName).SetName(name)
		return
	}
	m.name = &name
}

func (m *modifier) Username() string {
	if m.username != nil {
		return *m.username
	}
	return m.base.Username()
}

func (m *modifier) SetUsername(username string) {
	if m.base.(SettableUsername) != nil {
		m.base.(SettableUsername).SetUsername(username)
		return
	}
	m.username = &username
}

func (m *modifier) Password() string {
	if m.secret != nil {
		return *m.secret
	}
	return m.base.Password()
}

func (m *modifier) SetPassword(secret string) {
	if m.base.(SettablePassword) != nil {
		m.base.(SettablePassword).SetPassword(secret)
		return
	}
	m.secret = &secret
}

func (m *modifier) Type() string {
	if m.typ != nil {
		return *m.typ
	}
	return m.base.Type()
}

func (m *modifier) SetType(typ string) {
	if m.base.(SettableType) != nil {
		m.base.(SettableType).SetType(typ)
		return
	}
	m.typ = &typ
}

func (m *modifier) Fields() map[string]string {
	flds := maps.Merge(m.base.Fields(), m.fields)
	for _, f := range m.removeFields.Keys() {
		delete(flds, f)
	}
	return flds
}

func (m *modifier) GetField(name string) string {
	if m.fields != nil {
		if v, ok := m.fields[name]; ok {
			return v
		}
	}
	if m.removeFields != nil {
		if m.removeFields.Contains(name) {
			return ""
		}
	}
	return m.base.GetField(name)
}

func (m *modifier) SetField(name, value string) {
	if m.fields == nil {
		m.fields = map[string]string{}
	}
	m.fields[name] = value

	if m.removeFields != nil {
		m.removeFields.Delete(name)
	}
}

func (m *modifier) DeleteField(name string) {
	if m.fields != nil {
		delete(m.fields, name)
	}
	if m.removeFields == nil {
		m.removeFields = set.New[string]()
	}
	m.removeFields.Insert(name)
}

func (m *modifier) LastModified() time.Time {
	if m.lastModified != nil {
		return *m.lastModified
	}
	return m.base.LastModified()
}

func (m *modifier) SetLastModified(lastModified time.Time) {
	if m.base.(SettableLastModified) != nil {
		m.base.(SettableLastModified).SetLastModified(lastModified)
		return
	}
	m.lastModified = &lastModified
}

func (m *modifier) Url() *url.URL {
	if m.url != nil {
		return m.url
	}
	return m.base.Url()
}

func (m *modifier) SetUrl(url *url.URL) {
	if m.base.(SettableUrl) != nil {
		m.base.(SettableUrl).SetUrl(url)
		return
	}
	m.url = url
}

func (m *modifier) Location() string {
	if m.location != nil {
		return *m.location
	}
	return m.base.Location()
}

func SetName(secret Secret, name string) Secret {
	if mod, isMod := secret.(SettableName); isMod {
		mod.SetName(name)
		return secret
	}
	return &modifier{base: secret, name: &name}
}

func SetUsername(secret Secret, username string) Secret {
	if mod, isMod := secret.(SettableUsername); isMod {
		mod.SetUsername(username)
		return secret
	}
	return &modifier{base: secret, username: &username}
}

func SetSecret(secret Secret, secretValue string) Secret {
	if mod, isMod := secret.(SettablePassword); isMod {
		mod.SetPassword(secretValue)
		return secret
	}
	return &modifier{base: secret, secret: &secretValue}
}

func SetType(secret Secret, typ string) Secret {
	if mod, isMod := secret.(SettableType); isMod {
		mod.SetType(typ)
		return secret
	}
	return &modifier{base: secret, typ: &typ}
}

func SetField(secret Secret, name, value string) Secret {
	if mod, isMod := secret.(SettableFields); isMod {
		mod.SetField(name, value)
		return secret
	}

	return &modifier{base: secret, fields: map[string]string{name: value}}
}

func RemoveField(secret Secret, name string) Secret {
	if mod, isMod := secret.(SettableFields); isMod {
		mod.DeleteField(name)
		return secret
	}

	return &modifier{base: secret, removeFields: set.New(name)}
}

func SetLastModified(secret Secret, lastModified time.Time) Secret {
	if mod, isMod := secret.(SettableLastModified); isMod {
		mod.SetLastModified(lastModified)
		return secret
	}
	return &modifier{base: secret, lastModified: &lastModified}
}

func SetUrl(secret Secret, url *url.URL) Secret {
	if mod, isMod := secret.(SettableUrl); isMod {
		mod.SetUrl(url)
		return secret
	}
	return &modifier{base: secret, url: url}
}
