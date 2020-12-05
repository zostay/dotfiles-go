package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/ansd/lastpass-go"
)

var (
	LastPassUsername string
)

func init() {
	if u := os.Getenv("LPASS_USERNAME"); u != "" {
		LastPassUsername = u
	}
}

type LastPass struct {
	lp *lastpass.Client
}

func NewLastPass() (*LastPass, error) {
	u := LastPassUsername
	if LastPassUsername == "" {
		var err error
		u, err = PinEntry(
			"Zostay LastPass",
			"Asking for LastPass Username",
			"Username:",
			"OK",
		)
		if err != nil {
			return nil, err
		}
	}

	p, err := GetMasterPassword("LastPass", "LASTPASS-MASTER-"+u)
	if err != nil {
		return nil, err
	}

	lp, err := lastpass.NewClient(context.Background(), u, p)
	if err != nil {
		return nil, err
	}

	err = SetMasterPassword("LASTPASS-MASTER-"+u, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error keeping master password in memory.")
	}

	return &LastPass{lp}, nil
}

func (l *LastPass) GetSecret(name string) (string, error) {
	as, err := l.lp.Accounts(context.Background())
	if err != nil {
		return "", err
	}

	for _, a := range as {
		if a.Name == name {
			return a.Password, nil
		}
	}

	return "", ErrNotFound
}

func (l *LastPass) SetSecret(name, secret string) error {
	as, err := l.lp.Accounts(context.Background())
	if err != nil {
		return err
	}

	for _, a := range as {
		if a.Name == name {
			a.Password = secret
			err := l.lp.Update(context.Background(), a)
			return err
		}
	}

	a := lastpass.Account{
		Name:     name,
		Password: secret,
		Group:    ZostayRobotGroup,
	}

	err = l.lp.Add(context.Background(), &a)
	return err
}
