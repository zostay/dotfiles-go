package secrets

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/ansd/lastpass-go"
	"github.com/joho/godotenv"
)

const LocalEnv = ".zshrc.local" // where to find the LPASS_USERNAME

var (
	LastPassUsername string // the string loaded from LPASS_USERNAME
)

// init loads the .zshrc.local environment file and grabs the LPASS_USERNAME
// from it, which allows me to keep my LastPass username out of my dotfiles.
func init() {
	homedir, err := os.UserHomeDir()
	if err == nil {
		_ = godotenv.Load(path.Join(homedir, LocalEnv))
	}

	if u := os.Getenv("LPASS_USERNAME"); u != "" {
		LastPassUsername = u
	}
}

// LastPass is a secret Keeper that gets secrets from the LastPass
// password manager service.
type LastPass struct {
	lp *lastpass.Client
}

// NewLastPass constructs and returns a new LastPass Keeper or returns an error
// if there was a problem during construction.
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

// GetSecret returns the secret from the Lastpass service.
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

// SetSecret sets the secret into the LastPass service.
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
