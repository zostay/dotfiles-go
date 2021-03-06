package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd          *cobra.Command
	mailDir      string
	verbose      int
	dryRun       bool
	allowSending bool
)

func init() {
	cmd = &cobra.Command{
		Use:   "label-message <folder> <filename>",
		Short: "Sort a single email message",
		Run:   RunLabelMessage,
		Args:  cobra.ExactArgs(2),
	}

	cmd.PersistentFlags().StringVar(&mailDir, "maildir", mail.DefaultMailDir, "the root directory for mail")
	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	cmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "enable debugging verbose mode")
	cmd.PersistentFlags().BoolVarP(&allowSending, "allow-forwarding", "e", false, "allow email forwarding rules to run")
}

func RunLabelMessage(cmd *cobra.Command, args []string) {
	if mailDir == "" {
		panic(errors.New("maildir did not work"))
	}

	filter, err := mail.NewFilter(mailDir)
	if err != nil {
		panic(err)
	}

	filter.DryRun = dryRun
	filter.Debug = verbose

	folder := args[0]
	fn := args[1]

	actions, err := filter.LabelMessage(folder, fn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	fmt.Print(actions)
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
