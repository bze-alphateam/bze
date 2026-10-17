package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestDefaultParams_AreValid(t *testing.T) {
	require.Equal(t, types.NewParams(), types.DefaultParams())
	require.NoError(t, types.DefaultParams().Validate())
}
