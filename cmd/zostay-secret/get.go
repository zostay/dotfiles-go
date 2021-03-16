package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func initGet() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a secret",
		Args:  cobra.ExactArgs(1),
		RunE:  RunGetSecret,
	}

	cmd.AddCommand(getCmd)
}

func RunGetSecret(cmd *cobra.Command, args []string) error {
	k, err := secretKeeper()
	if err != nil {
		panic(err)
	}

	name := args[0]

	secret, err := k.GetSecret(name)
	if err != nil {
		panic(err)
	}
	fmt.Println(secret.Value)

	return nil
}
