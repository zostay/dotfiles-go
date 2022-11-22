package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMailDirFolder(t *testing.T) {
	mdf := NewMailDirFolder("test/maildir", "INBOX")
	assert.NotNil(t, mdf)

	assert.Equal(t, "test/maildir", mdf.Root())
	assert.Equal(t, "INBOX", mdf.Basename())
	assert.Equal(t, "test/maildir/INBOX", mdf.Path())
	assert.Equal(t, []string{
		"test/maildir/INBOX/new",
		"test/maildir/INBOX/cur",
	}, mdf.MessageDirPaths())
	assert.Equal(t, "test/maildir/INBOX/tmp", mdf.TempDirPath())
}

func TestDirFolder_Messages(t *testing.T) {
	mdf := NewMailDirFolder("test/maildir", "INBOX")
	assert.NotNil(t, mdf)

	ms, err := mdf.Messages()
	require.NoError(t, err)
	assert.Len(t, ms, 1)

	subj, err := ms[0].Subject()
	require.NoError(t, err)
	assert.Equal(t, "Foo", subj)
}

func TestDirFolder_Message(t *testing.T) {
	mdf := NewMailDirFolder("test/maildir", "INBOX")
	assert.NotNil(t, mdf)

	m, err := mdf.Message("1:2,S")
	require.NoError(t, err)

	subj, err := m.Subject()
	require.NoError(t, err)
	assert.Equal(t, "Foo", subj)
}
