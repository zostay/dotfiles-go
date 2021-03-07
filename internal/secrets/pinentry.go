package secrets

import (
	"fmt"
	"os"

	"github.com/gopasspw/gopass/pkg/pinentry"
)

// GetMasterPassword checks to see if the named master password is stored and
// available for retrieval. It returns it if it is. If it is not, it will popup
// a dialog box prompting the user to enter it.
func GetMasterPassword(which, name string) (string, error) {
	secret, err := Master.GetSecret(name)
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

// SetMasterPassword sets the named master password.
func SetMasterPassword(name, secret string) error {
	return Master.SetSecret(name, secret)
}

// PinEntry is a tool that makes it easier to display a dialog prompting the
// user for a password.
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
