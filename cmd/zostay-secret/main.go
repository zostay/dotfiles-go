// Package main provides an application for managing my personal secrets.
package main

import (
	"github.com/spf13/cobra"
)

var (
	cmd                                         *cobra.Command
	localOnly, remoteOnly, masterOnly, insecure bool
)

func init() {
	cmd = &cobra.Command{
		Use:   "zostay-secret",
		Short: "Work with my secrets",
	}

	cmd.PersistentFlags().BoolVarP(&localOnly, "local-only", "l", false, "only use the local database")
	cmd.PersistentFlags().BoolVarP(&remoteOnly, "remote-only", "r", false, "only use the remote database")
	cmd.PersistentFlags().BoolVarP(&masterOnly, "master", "m", false, "only use the system keyring")
	cmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "use the insecure store instead of the insecure store")

	initGet()
	initSet()
	initKeeper()
}

func main() {
	_ = cmd.Execute()
}
