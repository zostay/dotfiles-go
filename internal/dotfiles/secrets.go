package dotfiles

import (
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	ZostayGetSecretCommand = "bin/zostay-get-secret"
)

var (
	ZostayGetSecret string
)

func init() {
	var err error
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	ZostayGetSecret = path.Join(homedir, ZostayGetSecretCommand)
}

func GetSecret(name string) (string, error) {
	var s string

	c := exec.Command(ZostayGetSecret, name)

	obs, err := c.Output()
	if err != nil {
		return s, err
	}

	s = strings.TrimSpace(string(obs))

	return s, nil
}

func MustGetSecret(name string) string {
	s, err := GetSecret(name)
	if err != nil {
		panic(err)
	}
	return s
}
