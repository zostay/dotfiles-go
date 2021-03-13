// Package main provides an application for managing my personal secrets.
package main

import (
	"github.com/spf13/cobra"
)

var (
	cmd *cobra.Command
)

func init() {
	cmd = &cobra.Command{
		Use:   "zostay-secret",
		Short: "Work with my secrets",
	}
}

func main() {
	_ = cmd.Execute()
}
