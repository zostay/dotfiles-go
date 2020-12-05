package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

// encryption in here is probably just me being paranoid

type Internal struct {
	cipher  cipher.AEAD
	nonce   []byte
	secrets map[string][]byte
}

func MustNewInternal() *Internal {
	i, err := NewInternal()
	if err != nil {
		panic(err)
	}
	return i
}

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

func (i *Internal) GetSecret(name string) (string, error) {
	if s, ok := i.secrets[name]; ok {
		ds, err := i.cipher.Open(nil, i.nonce, s, nil)
		if err != nil {
			return "", err
		}
		return string(ds), nil
	} else {
		return "", ErrNotFound
	}
}

func (i *Internal) SetSecret(name, secret string) error {
	s := []byte(secret)
	es := i.cipher.Seal(nil, i.nonce, s, nil)
	i.secrets[name] = es
	return nil
}
