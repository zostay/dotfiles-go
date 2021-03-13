package secrets

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

const (
	ZostaySecretsFile = ".secrets.yaml" // where to store low security secrets
)

var (
	ZostaySecretsPath string // the path to the low secrutiy secrets
)

// init sets up ZostaySecretsPath
func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostaySecretsPath = path.Join(homedir, ZostaySecretsFile)
}

// LowSecurity is a secret Keeper that stores secrets in plain text. There are a
// few secrets are used in such a way that no additional security is required.
type LowSecurity struct{}

// loadSecrets loads the secrets from file.
func (*LowSecurity) loadSecrets() (map[string]string, error) {
	yamlSecrets, err := ioutil.ReadFile(ZostaySecretsPath)
	if err != nil {
		return nil, err
	}

	var secrets map[string]string
	err = yaml.Unmarshal(yamlSecrets, &secrets)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

// saveSecrets saves the secrets to file.
func (*LowSecurity) saveSecrets(s map[string]string) error {
	out, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ZostaySecretsPath, out, 0644)
	if err != nil {
		return err
	}

	return nil
}

// GetSecret retrieves the named secret.
func (s *LowSecurity) GetSecret(name string) (*Secret, error) {
	secrets, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	if s, ok := secrets[name]; ok {
		return &Secret{
			Name:  name,
			Value: s,
		}, nil
	}

	return nil, ErrNotFound
}

// SetSecret saves the named secret.
func (s *LowSecurity) SetSecret(secret *Secret) error {
	secrets, err := s.loadSecrets()
	if err != nil {
		return err
	}

	secrets[secret.Name] = secret.Value

	err = s.saveSecrets(secrets)
	if err != nil {
		return err
	}

	return nil
}

func (*LowSecurity) RemoveSecret(name string) error {
	return errors.New("not implemented")
}
