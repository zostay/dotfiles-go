package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zostay/go-addr/pkg/addr"
)

func TestAddressListStrings(t *testing.T) {
	t.Parallel()

	addrs, err := addr.ParseEmailAddressList("sterling@example.com, Foo@example.com, blahblah@example.com")
	require.NoError(t, err)

	strs := AddressListStrings(addrs)

	assert.Equal(t, []string{
		"sterling@example.com",
		"Foo@example.com",
		"blahblah@example.com",
	}, strs)
}

func TestAddressListHTML(t *testing.T) {
	t.Parallel()

	addrs, err := addr.ParseEmailAddressList("sterling@example.com, Foo@example.com, blahblah@example.com")
	require.NoError(t, err)

	str := AddressListHTML(addrs)
	assert.Equal(t, "<strong></strong> &lt;<a href=\"mailto:sterling@example.com\">sterling@example.com</a>&gt;, <strong></strong> &lt;<a href=\"mailto:Foo@example.com\">Foo@example.com</a>&gt;, <strong></strong> &lt;<a href=\"mailto:blahblah@example.com\">blahblah@example.com</a>&gt;", str)
}
