package dotfiles

import (
	"os/exec"
	"path"
	"strings"
)

var (
	ZostayGetSecret = path.Join(HomeDir, "bin/zostay-get-secret")
)

func GetSecret(name string) (string, error) {
	var s string

	c := exec.Command(ZostayGetSecret, name)
	err := c.Run()
	if err != nil {
		return s, err
	}

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
