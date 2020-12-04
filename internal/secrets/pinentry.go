package secrets

import (
	"fmt"
	"os"

	"github.com/gopasspw/gopass/pkg/pinentry"
)

var master = NewHttp()

func GetMasterPassword(which, name string) (string, error) {
	secret, err := master.GetSecret(name)
	if err == nil {
		return secret, nil
	} else {
		fmt.Fprintf(os.Stderr, "Unable to retrieve password from secret keeper daemon: %v\n", err)
		fmt.Fprintln(os.Stderr, "Fallback to pinentry.")
	}

	return PinEntry(
		"Zostay "+which,
		"Asking for "+which+" Password",
		"Password:",
		"OK",
	)
}

func SetMasterPassword(name, secret string) error {
	return master.SetSecret(name, secret)
}

func PinEntry(title, desc, prompt, ok string) (string, error) {
	pi, err := pinentry.New()
	if err != nil {
		return "", err
	}

	_ = pi.Set("title", title)
	_ = pi.Set("desc", desc)
	_ = pi.Set("prompt", prompt)
	_ = pi.Set("ok", ok)
	x, err := pi.GetPin()
	if err != nil {
		return "", err
	}

	return string(x), nil
}
