package main

import (
	"github.com/spf13/cobra"

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

	addr := make(mail.AddressList, 1)
	addr[0] = &mail.Address{Address: to}

	err := m.ForwardTo(addr...)
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
