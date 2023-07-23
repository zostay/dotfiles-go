package low

import "github.com/zostay/go-std/maps"

type lowSecurityConfig struct {
	Version string
	Secrets map[string]Secret
}

func newLowSecurityConfig() *lowSecurityConfig {
	return &lowSecurityConfig{
		Version: "1",
	}
}

func (c *lowSecurityConfig) get(id string) (*Secret, bool) {
	if c.Secrets == nil {
		return &Secret{}, false
	}

	sec := c.Secrets[id]
	sec.SetID(id)
	return &sec, true
}

func (c *lowSecurityConfig) set(secret Secret) {
	if c.Secrets == nil {
		c.Secrets = make(map[string]Secret, 1)
	}

	c.Secrets[secret.ID()] = secret
}

func (c *lowSecurityConfig) delete(id string) {
	if c.Secrets == nil {
		return
	}

	delete(c.Secrets, id)
}

func (c *lowSecurityConfig) iterator() *maps.Iterator[string, Secret] {
	return maps.NewIterator(c.Secrets)
}
