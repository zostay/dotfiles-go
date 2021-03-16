package dotfiles

import (
	"errors"
	// "github.com/zostay/dotfiles-go/internal/secrets"
)

func GetSecret(name string) (string, error) {
	return "", errors.New("not implemented")
	// return secrets.AutoKeeper().GetSecret(name)
}

func MustGetSecret(name string) string {
	panic("not implemented")
	// s, err := GetSecret(name)
	// if err != nil {
	// 	panic(err)
	// }
	// return s
}
