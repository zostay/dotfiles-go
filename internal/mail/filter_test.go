package mail

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkFilter(t *testing.T) *Filter {
	f, err := NewFilter("test/maildir", "test/rules.yaml", "test/local.yaml")
	require.NoError(t, err)
	return f
}

func TestNewFilter(t *testing.T) {
	_, err := NewFilter("test/maildir", "test/nonexistent-rules.yaml", "test/local.yaml")
	assert.Error(t, err)

	_, err = NewFilter("test/maildir", "test/rules.yaml", "test/nonexistent-local.yaml")
	assert.Error(t, err)

	f, err := NewFilter("test/maildir", "test/rules.yaml", "test/local.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, f)
}

func TestFilter_LimitSince(t *testing.T) {
	f := mkFilter(t)

	f.UseNow(time.Date(2022, 11, 22, 23, 11, 59, 0, time.Local))
	f.LimitFilterToRecent(48 * time.Hour)

	since := f.LimitSince()
	assert.Equal(t, time.Date(2022, 11, 20, 23, 11, 59, 0, time.Local), since)
}

func TestFilter_Message(t *testing.T) {
	f := mkFilter(t)

	msg, err := f.Message("INBOX", "1:2,S")
	require.NoError(t, err)

	subj, err := msg.Subject()
	assert.NoError(t, err)
	assert.Equal(t, "Foo", subj)

	msg, err = f.Message("Other", "2:2,S")
	require.NoError(t, err)

	subj, err = msg.Subject()
	assert.NoError(t, err)
	assert.Equal(t, "Bar", subj)
}

func TestFilter_Messages(t *testing.T) {
	f := mkFilter(t)

	msgs, err := f.Messages("INBOX")
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	subj, err := msgs[0].Subject()
	assert.NoError(t, err)
	assert.Equal(t, "Foo", subj)

	msgs, err = f.Messages("Other")
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	var subjs [2]string

	for i, msg := range msgs {
		subj, err := msg.Subject()
		assert.NoError(t, err)
		subjs[i] = subj
	}

	sort.Strings(subjs[:])

	assert.Equal(t, "Bar", subjs[0])
	assert.Equal(t, "Baz", subjs[1])
}

func TestFilter_AllFolders(t *testing.T) {
	f := mkFilter(t)

	folders, err := f.AllFolders()
	require.NoError(t, err)

	assert.Len(t, folders, 2)

	sort.Strings(folders)
	assert.Equal(t, []string{"INBOX", "Other"}, folders)
}
