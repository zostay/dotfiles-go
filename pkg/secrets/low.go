package secrets

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"

	"github.com/zostay/dotfiles-go/internal/fssafe"
)

// LowSecurity is a secret Keeper that stores secrets in plain text. There are a
// few secrets are used in such a way that no additional security is required.
type LowSecurity struct {
	fssafe.LoaderSaver
}

// NewLowSecurity creates a low security secret store at the given path.
func NewLowSecurity(path string) *LowSecurity {
	return &LowSecurity{fssafe.NewFileSystemLoaderSaver(path)}
}

// loadSecrets loads the secrets from file.
func (s *LowSecurity) loadSecrets() (map[string]string, error) {
	r, err := s.Loader()
	if err != nil {
		// give saving a try first
		_ = s.saveSecrets(map[string]string{})
		r, err = s.Loader()
		if err != nil {
			return nil, err
		}
	}

	yamlSecrets, err := ioutil.ReadAll(r)
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
func (s *LowSecurity) saveSecrets(ss map[string]string) error {
	out, err := yaml.Marshal(ss)
	if err != nil {
		return err
	}

	w, err := s.Saver()
	if err != nil {
		return err
	}

	_, _ = w.Write(out)
	err = w.Close()
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

func (s *LowSecurity) RemoveSecret(name string) error {
	secrets, err := s.loadSecrets()
	if err != nil {
		return err
	}

	delete(secrets, name)

	err = s.saveSecrets(secrets)
	if err != nil {
		return err
	}

	return nil
}
