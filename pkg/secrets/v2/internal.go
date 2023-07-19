package secrets

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"

	"github.com/oklog/ulid/v2"
)

// encryption in here is probably just me being paranoid

// Internal is a Keeper that stores secrets in memory.
type Internal struct {
	cipher  cipher.AEAD
	nonce   []byte
	secrets map[string][]byte
}

var _ Keeper = &Internal{}

// MustNewInternal calls NewInternal and panics if it returns an error.
func MustNewInternal() *Internal {
	i, err := NewInternal()
	if err != nil {
		panic(err)
	}
	return i
}

// NewInternal constructs a new secret memory store.
func NewInternal() (*Internal, error) {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	gc, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gc.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	i := &Internal{
		cipher:  gc,
		nonce:   nonce,
		secrets: make(map[string][]byte),
	}

	return i, nil
}

func (i *Internal) decodeSecret(s []byte) (Secret, error) {
	ds, err := i.cipher.Open(nil, i.nonce, s, nil)
	if err != nil {
		return nil, err
	}

	dec := gob.NewDecoder(bytes.NewReader(ds))

	var sec Single
	err = dec.Decode(&sec)
	if err != nil {
		return nil, err
	}

	return &sec, nil
}

func (i *Internal) encodeSecret(sec Secret) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(sec)
	if err != nil {
		return nil, err
	}

	s := buf.Bytes()
	es := i.cipher.Seal(nil, i.nonce, s, nil)

	return es, nil
}

// ListLocations returns a list of all the secret names in the store.
func (i *Internal) ListLocations(context.Context) ([]string, error) {
	locs := make([]string, 0, len(i.secrets)>>1)
	for _, ct := range i.secrets {
		sec, err := i.decodeSecret(ct)
		if err != nil {
			return nil, err
		}

		locs = append(locs, sec.Location())
	}
	return locs, nil
}

// ListSecrets returns a list of all the secret IDs at the given location.
func (i *Internal) ListSecrets(_ context.Context, loc string) ([]string, error) {
	ids := make([]string, 0, len(i.secrets)>>1)
	for _, ct := range i.secrets {
		sec, err := i.decodeSecret(ct)
		if err != nil {
			return nil, err
		}

		if sec.Location() == loc {
			ids = append(ids, sec.ID())
		}
	}
	return ids, nil
}

// GetSecret retrieves the identified secret from the internal memory store.
func (i *Internal) GetSecret(_ context.Context, id string) (Secret, error) {
	if s, ok := i.secrets[id]; ok {
		return i.decodeSecret(s)
	}
	return nil, ErrNotFound
}

// GetSecretsByName retrieves all secrets with the given name.
func (i *Internal) GetSecretsByName(_ context.Context, name string) ([]Secret, error) {
	secs := make([]Secret, 0, 1)
	for _, ct := range i.secrets {
		sec, err := i.decodeSecret(ct)
		if err != nil {
			return nil, err
		}

		if sec.Name() == name {
			secs = append(secs, sec)
		}
	}
	return secs, nil
}

// SetSecret saves the named secret to the given value in the internal memory
// store.
func (i *Internal) SetSecret(_ context.Context, secret Secret) (Secret, error) {
	single := NewSingleFromSecret(secret)
	if _, hasSecret := i.secrets[single.id]; single.id == "" || !hasSecret {
		single.id = ulid.Make().String()
	}

	es, err := i.encodeSecret(single)
	if err != nil {
		return nil, err
	}

	i.secrets[single.ID()] = es
	return single, nil
}

// CopySecret copies the secret into a new location while leaving the original
// in the old location.
func (i *Internal) CopySecret(ctx context.Context, id string, location string) (Secret, error) {
	secret, err := i.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}

	cp := NewSingleFromSecret(secret)
	cp.location = location
	cp.id = ulid.Make().String()

	es, err := i.encodeSecret(secret)
	if err != nil {
		return nil, err
	}

	i.secrets[cp.ID()] = es
	return cp, nil
}

// MoveSecret moves the secret into another location of the memory store.
func (i *Internal) MoveSecret(ctx context.Context, id string, location string) (Secret, error) {
	secret, err := i.GetSecret(ctx, id)
	if err != nil {
		return nil, err
	}

	mv := NewSingleFromSecret(secret)
	mv.location = location

	es, err := i.encodeSecret(secret)
	if err != nil {
		return nil, err
	}

	i.secrets[mv.ID()] = es
	return mv, nil
}

// DeleteSecret removes the identified secret from the store.
func (i *Internal) DeleteSecret(_ context.Context, id string) error {
	delete(i.secrets, id)
	return nil
}
