package secrets

import (
	"fmt"
	"os"
)

type Cacher struct {
	source Keeper
	target Keeper
}

func NewCacher(src, target Keeper) *Cacher {
	return &Cacher{src, target}
}

func (c *Cacher) GetSecret(name string) (string, error) {
	s, err := c.target.GetSecret(name)
	if err == ErrNotFound {
		s, err := c.source.GetSecret(name)
		if err != nil {
			return s, err
		}

		err = c.target.SetSecret(name, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error caching secret %s locally: %v\n", name, err)
			return s, nil
		}
	} else if err != nil {
		return "", err
	}

	return s, nil
}

func (c *Cacher) SetSecret(name, secret string) error {
	err1 := c.target.SetSecret(name, secret)
	err2 := c.source.SetSecret(name, secret)

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}
