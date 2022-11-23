package main

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/zostay/go-addr/pkg/addr"

	"github.com/zostay/dotfiles-go/internal/mail"
)

var (
	cmd *cobra.Command
	to  string
	msg string
)

func init() {
	cmd = &cobra.Command{
		Use:   "forward-file",
		Short: "Forward a message file",
		Run:   RunForward,
	}

	cmd.PersistentFlags().StringVarP(&to, "to", "t", "", "email address to receive the forward")
	cmd.PersistentFlags().StringVarP(&msg, "file", "f", "", "the file name of the message to forward")
}

func RunForward(cmd *cobra.Command, args []string) {
	m := mail.NewFileMessage(msg)

	as := make(addr.AddressList, 1)
	var err error
	as[0], err = addr.NewMailboxStr("", to, "")
	if err != nil {
		panic(err)
	}

	err = m.ForwardTo(as, time.Now())
	if err != nil {
		panic(err)
	}
}

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
