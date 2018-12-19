package sync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefix(t *testing.T) {
	require.Equal(t, []string{"pre-foo", "pre-bar"}, prefix("pre-", []string{"foo", "bar"}))
}
