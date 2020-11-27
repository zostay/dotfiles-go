package mail

import (
	"html"

	"github.com/emersion/go-message/mail"
)

type Address = mail.Address

type AddressList []*Address

func AddressListStrings(addr AddressList) []string {
	ss := make([]string, len(addr))
	for i, a := range addr {
		ss[i] = a.Address
	}
	return ss
}

func AddressListString(addr AddressList) string {
	var addrStr string
	for _, a := range addr {
		astr := a.String()
		if addrStr != "" {
			addrStr += ", "
		}
		addrStr += astr
	}
	return addrStr
}

func AddressListHTML(addr AddressList) string {
	var addrStr string
	for _, a := range addr {
		astr := "<strong>" + html.EscapeString(a.Name) + "</strong> &lt;<a href=\"mailto:"
		astr += html.EscapeString(a.Address) + "\">"
		astr += html.EscapeString(a.Address) + "</a>&gt;"
		if addrStr != "" {
			addrStr += ", "
		}
		addrStr += astr
	}
	return addrStr
}
