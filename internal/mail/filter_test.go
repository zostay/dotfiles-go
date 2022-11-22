package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFilter(t *testing.T) {
	_, err := NewFilter("test/maildir", "test/nonexistent-rules.yaml", "test/local.yaml")
	assert.Error(t, err)

	_, err = NewFilter("test/maildir", "test/rules.yaml", "test/nonexistent-local.yaml")
	assert.Error(t, err)

	f, err := NewFilter("test/maildir", "test/rules.yaml", "test/local.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, f)
}
