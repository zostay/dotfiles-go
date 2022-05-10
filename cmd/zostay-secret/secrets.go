package main

import (
	"errors"

	"github.com/zostay/dotfiles-go/internal/keeper"
	"github.com/zostay/dotfiles-go/pkg/secrets"
)

// secretKeeper sets up all the secrets SDK, interprets the command-line
// options, chooses or builds a secret keeper based on those options, and
// returns a secret keeper.
//
// The behavior is as follows:
//
// If more than one secret keeper selection option is set
// (--master/--local-only/--remove-only), then an error is returned.
//
// Otherwise, if --master is set, then this returns
//
//  secrets.Master()
//
// If --remote-only is set, this returns either secrets.SecureMain() or
// secrets.InsecureMain() based on the --insecure option.
//
// If --local-only is set, this returns either secrets.SecureLocal() or
// secrets.InsecureLocal() based on the --insecure option.
//
// Finally, if none of secret keeper selection options are set, it returns
// either secrets.Secure() or secrets.Insecure() based on --insecure option.
func secretKeeper() (secrets.Keeper, error) {
	keeper.RequiresSecretKeeper()

	if localOnly && remoteOnly || localOnly && masterOnly || remoteOnly && masterOnly {
		return nil, errors.New("only one of these options may be specified: --local-only/-l, --remote-only/-r, --master/-m")
	}

	var k secrets.Keeper
	if masterOnly {
		var err error
		k, err = secrets.Master()
		if err != nil {
			panic(err)
		}
	} else if remoteOnly {
		var err error
		if insecure {
			k, err = secrets.InsecureMain()
		} else {
			k, err = secrets.SecureMain()
		}

		if err != nil {
			panic(err)
		}
	} else if localOnly {
		var err error
		if insecure {
			k, err = secrets.InsecureLocal()
		} else {
			k, err = secrets.SecureLocal()
		}

		if err != nil {
			panic(err)
		}
	} else {
		var err error
		if insecure {
			k, err = secrets.Insecure()
		} else {
			k, err = secrets.Secure()
		}

		if err != nil {
			panic(err)
		}
	}

	return k, nil
}
