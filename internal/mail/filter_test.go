package mail

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkFilter(t *testing.T) *Filter {
	f, err := NewFilter("test/maildir", "test/rules.yml", "test/local.yml")
	f.UseNow(time.Date(2022, 11, 22, 23, 11, 59, 0, time.Local))
	require.NoError(t, err)
	return f
}

func mkFilterDR(t *testing.T) *Filter {
	f := mkFilter(t)
	f.SetDryRun(true)
	return f
}

func TestNewFilter(t *testing.T) {
	_, err := NewFilter("test/maildir", "test/nonexistent-rules.yml", "test/local.yml")
	assert.Error(t, err)

	_, err = NewFilter("test/maildir", "test/rules.yml", "test/nonexistent-local.yml")
	assert.Error(t, err)

	f, err := NewFilter("test/maildir", "test/rules.yml", "test/local.yml")
	assert.NoError(t, err)
	assert.NotNil(t, f)
}

func TestFilter_LimitSince(t *testing.T) {
	f := mkFilter(t)

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

	var msg Message
	ok := msgs.Next(&msg)
	assert.True(t, ok)

	subj, err := msg.Subject()
	assert.NoError(t, err)
	assert.Equal(t, "Foo", subj)

	ok = msgs.Next(&msg)
	assert.False(t, ok)
	assert.NoError(t, msgs.Err())

	msgs, err = f.Messages("Other")
	require.NoError(t, err)

	var subjs [2]string

	i := 0
	for msgs.Next(&msg) {
		subj, err := msg.Subject()
		assert.NoError(t, err)
		subjs[i] = subj
		i++
	}

	assert.NoError(t, msgs.Err())

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

func TestFilter_RulesForFolder(t *testing.T) {
	f := mkFilter(t)

	rules := f.RulesForFolder("INBOX")
	require.Len(t, rules, 1)

	assert.Equal(t, &CompiledRule{
		Match: Match{
			Folder: "INBOX",
			Days:   7,
			From:   "sterling@example.com",
		},
		Label:    []string{"Other"},
		OkayDate: time.Date(2022, 11, 15, 23, 11, 59, 0, time.Local),
	}, rules[0])

	rules = f.RulesForFolder("Other")
	require.Len(t, rules, 1)

	assert.Equal(t, &CompiledRule{
		Match: Match{
			Folder: "Other",
			Days:   10,
		},
		Clear:    []string{`\Inbox`},
		OkayDate: time.Date(2022, 11, 12, 23, 11, 59, 0, time.Local),
	}, rules[0])
}

func TestFilter_LabelMessage(t *testing.T) {
	f := mkFilterDR(t)

	actions, err := f.LabelMessage("INBOX", "1:2,S")
	assert.NoError(t, err)

	assert.Equal(t, ActionsSummary{
		"Labeled Other": 1,
	}, actions)
}

func TestFilter_LabelMessages(t *testing.T) {
	f := mkFilterDR(t)

	actions, err := f.LabelMessages([]string{"INBOX"})
	assert.NoError(t, err)

	assert.Equal(t, ActionsSummary{
		"Labeled Other": 1,
	}, actions)
}
