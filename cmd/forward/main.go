package main

import (
	email "github.com/emersion/go-message/mail"

	"github.com/zostay/dotfiles-go/internal/mail"
)

func main() {
	// r, err := os.Open("Test/INBOX/cur/1602005307_1.12.f3d93a06c131,U=101733,FMD5=7e33429f656f1e6e9d79b29c3f82c57e:2,S")
	// if err != nil {
	// 	panic(err)
	// }

	// e, err := message.Read(r)
	// if err != nil {
	// 	panic(err)
	// }

	// mr := email.NewReader(e)
	// mr, err := email.CreateReader(r)
	// if err != nil {
	// 	panic(err)
	// }

	// for {
	// 	pr, err := mr.NextPart()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	switch ph := pr.Header.(type) {
	// 	case *email.InlineHeader:
	// 		fmt.Println(strings.Repeat("INLINE ", 10))
	// 		io.Copy(os.Stdout, pr.Body)
	// 		fmt.Println(strings.Repeat("END ", 20))
	// 	case *email.AttachmentHeader:
	// 		fn, err := ph.Filename()
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		fmt.Println("ATTACHMENT: " + fn)
	// 	}
	// }

	m := mail.NewMessage(
		"Test/INBOX",
		"1602005307_1.12.f3d93a06c131,U=101733,FMD5=7e33429f656f1e6e9d79b29c3f82c57e",
	)

	addr := make([]*email.Address, 1)
	addr[0] = &email.Address{Name: "ASH", Address: "sterling@hanenkamp.com"}

	err := m.ForwardTo(addr...)
	if err != nil {
		panic(err)
	}

	// r, err := m.ForwardReader(addr)
	// if err != nil {
	// 	panic(err)
	// }

	// _, err = io.Copy(os.Stdout, r)
	// if err != nil {
	// 	panic(err)
	// }
}
