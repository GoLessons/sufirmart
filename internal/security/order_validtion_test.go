package security

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrderNumValidation_Basic(t *testing.T) {
	require.True(t, OrderNumValidation("79927398713"))
	require.False(t, OrderNumValidation("79927398714"))
	require.False(t, OrderNumValidation("qwerty"))
	require.False(t, OrderNumValidation(""))
}
