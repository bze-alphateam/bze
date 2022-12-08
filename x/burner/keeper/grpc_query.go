package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
)

var _ types.QueryServer = Keeper{}
