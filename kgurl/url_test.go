package kgurl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	u, err := Parse("/coffee")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "/coffee", u.Path)
}
