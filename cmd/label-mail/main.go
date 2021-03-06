package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
	_ "github.com/zostay/go-addr/pkg/addr/encoding"
	_ "github.com/zostay/go-email/pkg/email/encoding"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd          *cobra.Command
	allMail      bool
	mailDir      string
	dryRun       bool
	verbose      int
	folders      []string
	allowSending bool
	cpuprofile   string
)

func init() {
	cmd = &cobra.Command{
		Use:   "label-mail",
		Short: "Sort my email in the local MailDir",
		Run:   RunLabelMail,
	}

	cmd.PersistentFlags().BoolVarP(&allMail, "all-mail", "a", false, "run against mail from all time")
	cmd.PersistentFlags().StringVar(&mailDir, "maildir", mail.DefaultMailDir, "the root directory for mail")
	cmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "perform a dry run")
	cmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "enable debugging verbose mode")
	cmd.PersistentFlags().StringSliceVarP(&folders, "folder", "f", []string{}, "select folders to filter")
	cmd.PersistentFlags().BoolVarP(&allowSending, "allow-forwarding", "e", false, "allow email forwarding rules to run")
	cmd.PersistentFlags().StringVar(&cpuprofile, "cpuprofile", "", "write CPU profile to `file`")
}

func RunLabelMail(cmd *cobra.Command, args []string) {
	if mailDir == "" {
		panic(errors.New("maildir did not work"))
	}

	filter, err := mail.NewFilter(mailDir)
	if err != nil {
		panic(err)
	}

	if !allMail {
		filter.LimitFilterToRecent(2 * time.Hour)
	}

	filter.DryRun = dryRun
	filter.Debug = verbose

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	actions, err := filter.LabelMessages(folders)
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
