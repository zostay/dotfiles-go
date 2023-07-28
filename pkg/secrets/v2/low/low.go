package low

import (
	"context"
	"errors"
	"io/ioutil"
	"strconv"

	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"

	"github.com/zostay/dotfiles-go/internal/fssafe"
	"github.com/zostay/dotfiles-go/pkg/secrets/v2"
)

var ErrUnsupportedVersion = errors.New("unsupported low security file version")

// LowSecurity is a secret keeper for storing secrets in plain text. Only
// suitable for very low value secrets.
type LowSecurity struct {
	// LoaderSaver is the loader/saver to use for the secrets.
	fssafe.LoaderSaver
}

var _ secrets.Keeper = &LowSecurity{}

// NewLowSecurity creates a new low security secret keeper at the given path.
func NewLowSecurity(path string) *LowSecurity {
	return &LowSecurity{
		LoaderSaver: fssafe.NewFileSystemLoaderSaver(path),
	}
}

// NewLowSecurityCustom creates a new low security secret keeper with the given
// loader/saver.
func NewLowSecurityCustom(ls fssafe.LoaderSaver) *LowSecurity {
	return &LowSecurity{
		LoaderSaver: ls,
	}
}

func makeID() string {
	return ulid.MustNew(ulid.Now(), nil).String()
}

// loadSecrets loads the secrets from file.
func (s *LowSecurity) loadSecrets() (*lowSecurityConfig, error) {
	r, err := s.Loader()
	if err != nil {
		// give saving a try first
		_ = s.saveSecrets(newLowSecurityConfig())
		r, err = s.Loader()
		if err != nil {
			return nil, err
		}
	}

	yamlSecrets, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var cfg lowSecurityConfig
	err = yaml.Unmarshal(yamlSecrets, &cfg)
	if err != nil {
		return nil, err
	}

	version, err := strconv.Atoi(cfg.Version)
	if err != nil {
		version = 1
	}

	if version > 1 || version < 1 {
		return nil, ErrUnsupportedVersion
	}

	return &cfg, nil
}

// saveSecrets saves the secrets to file.
func (s *LowSecurity) saveSecrets(cfg *lowSecurityConfig) error {
	w, err := s.Saver()
	if err != nil {
		return err
	}

	yamlSecrets, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	_, err = w.Write(yamlSecrets)
	if err != nil {
		return err
	}

	return nil
}

// ListLocations returns all the locations listed in the low security file.
func (s *LowSecurity) ListLocations(context.Context) ([]string, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(cfg.Secrets))
	iter := cfg.iterator()
	for iter.Next() {
		ids = append(ids, iter.ID())
	}

	return ids, nil
}

// ListSecrets returns all the secrets listed in the low security file for the
// given location.
func (s *LowSecurity) ListSecrets(
	ctx context.Context,
	location string,
) ([]string, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(cfg.Secrets))
	iter := cfg.iterator()
	if iter.Next() {
		secret := iter.Val()
		if secret.Location() == location {
			ids = append(ids, iter.ID())
		}
	}

	return ids, nil
}

// GetSecret returns the secret with the given ID from the low security file.
func (s *LowSecurity) GetSecret(
	ctx context.Context,
	id string,
) (secrets.Secret, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	sec, hasSecret := cfg.get(id)
	if !hasSecret {
		return nil, secrets.ErrNotFound
	}

	return sec, nil
}

// GetSecretsByName returns all the secrets with the given name from the low
// security file.
func (s *LowSecurity) GetSecretsByName(
	ctx context.Context,
	name string,
) ([]secrets.Secret, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	secs := make([]secrets.Secret, 0, len(cfg.Secrets))
	iter := cfg.iterator()
	for iter.Next() {
		secret := iter.Val()
		single := secrets.NewSingleFromSecret(&secret)
		sec := &Secret{Single: *single}
		if secret.Name() == name {
			secs = append(secs, sec)
		}
	}

	return secs, nil
}

// SetSecret sets the secret with the given ID in the low security file.
func (s *LowSecurity) SetSecret(
	ctx context.Context,
	secret secrets.Secret,
) (secrets.Secret, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	single := secrets.NewSingleFromSecret(secret)
	sec := Secret{Single: *single}

	sec.SetID(makeID())
	cfg.set(sec)

	err = s.saveSecrets(cfg)
	if err != nil {
		return nil, err
	}

	return &sec, nil
}

// DeleteSecret deletes the secret with the given ID from the low security
// file.
func (s *LowSecurity) DeleteSecret(
	ctx context.Context,
	id string,
) error {
	cfg, err := s.loadSecrets()
	if err != nil {
		return err
	}

	cfg.delete(id)

	err = s.saveSecrets(cfg)
	if err != nil {
		return err
	}

	return nil
}

// CopySecret copies the secret with the given ID from the low security file
// to the given location.
func (s *LowSecurity) CopySecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	sec, hasSecret := cfg.get(id)
	if !hasSecret {
		return nil, secrets.ErrNotFound
	}

	sec.SetID(makeID())
	sec.SetLocation(location)
	cfg.set(*sec)

	err = s.saveSecrets(cfg)
	if err != nil {
		return nil, err
	}

	return sec, nil
}

// MoveSecret moves the secre to a new location.
func (s *LowSecurity) MoveSecret(
	ctx context.Context,
	id string,
	location string,
) (secrets.Secret, error) {
	cfg, err := s.loadSecrets()
	if err != nil {
		return nil, err
	}

	sec, hasSecret := cfg.get(id)
	if !hasSecret {
		return nil, secrets.ErrNotFound
	}

	sec.SetLocation(location)
	cfg.set(*sec)

	err = s.saveSecrets(cfg)
	if err != nil {
		return nil, err
	}

	return sec, nil
}
