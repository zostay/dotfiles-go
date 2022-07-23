package mail

import (
	"html"

	"github.com/zostay/go-addr/pkg/addr"
)

// AddressListStrings a list of strings created by calling CleanString() on each
// address in the given addr.AddressList.
func AddressListStrings(as addr.AddressList) []string {
	ss := make([]string, len(as))
	for i, a := range as {
		ss[i] = a.CleanString()
	}
	return ss
}

// AddressListHTML returns an HTML representation of the addr.AddressList.
func AddressListHTML(addr addr.AddressList) string {
	var addrStr string
	for _, a := range addr {
		astr := "<strong>" + html.EscapeString(a.DisplayName()) + "</strong> &lt;<a href=\"mailto:"
		astr += html.EscapeString(a.Address()) + "\">"
		astr += html.EscapeString(a.Address()) + "</a>&gt;"
		if addrStr != "" {
			addrStr += ", "
		}
		addrStr += astr
	}
	return addrStr
}
