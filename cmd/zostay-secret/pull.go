package main

import (
	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/secrets"
)

func RunSecretPull(cmd *cobra.Command, args []string) {
	RequiresSecretKeeper()

	lp, err := secrets.NewLastPass()
	if err != nil {
		panic(err)
	}

	kp, err := secrets.NewKeepass()
	if err != nil {
		panic(err)
	}

	name := args[0]

	s, err := lp.GetSecret(name)
	if err != nil {
		panic(err)
	}

	err = kp.SetSecret(name, s)
	if err != nil {
		panic(err)
	}
}
