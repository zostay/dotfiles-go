package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd     *cobra.Command
	allMail bool
	mailDir string
	dryRun  bool
	verbose int
	folders []string
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

	actions, err := filter.LabelMessages(folders)
	if err != nil {
		panic(err)
	}

	total := 0
	kw := 5
	cw := 1
	keys := make([]string, 0, len(actions))
	for key, count := range actions {
		total += count
		keys = append(keys, key)

		if len(key) > kw {
			kw = len(key)
		}

		countLen := len(strconv.Itoa(count))
		if countLen > cw {
			cw = countLen
		}
	}

	sort.Strings(keys)

	kws := strconv.Itoa(kw)
	cws := strconv.Itoa(cw)

	if total > 0 {
		for _, key := range keys {
			fmt.Printf(" %-"+kws+"s : %"+cws+"d\n", key, actions[key])
		}

		fmt.Printf("%s %s\n", strings.Repeat("-", kw+2), strings.Repeat("-", cw+2))
		fmt.Printf(" %-"+kws+"s : %"+cws+"d\n", "Total", total)
	} else {
		fmt.Println("Nothing to do.")
	}
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
