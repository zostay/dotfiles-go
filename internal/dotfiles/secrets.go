package dotfiles

import (
	"github.com/zostay/dotfiles-go/internal/secrets"
)

func GetSecret(name string) (string, error) {
	return secrets.AutoKeeper().GetSecret(name)
}

func MustGetSecret(name string) string {
	s, err := GetSecret(name)
	if err != nil {
		panic(err)
	}
	return s
}
