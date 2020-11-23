package dotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const (
	EnvFile = ".dotfile-environment"
)

var (
	HomeDir string
)

func init() {
	var err error
	HomeDir, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
}

func SetEnvironment(env string) error {
	fh, err := os.Create(path.Join(HomeDir, EnvFile))
	if err != nil {
		return err
	}

	defer fh.Close()

	fmt.Fprintln(fh, env)

	return nil
}

func Environment() (string, error) {
	bs, err := ioutil.ReadFile(path.Join(HomeDir, EnvFile))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(bs)), err
}
