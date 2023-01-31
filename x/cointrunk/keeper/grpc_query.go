package keeper

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

var _ types.QueryServer = Keeper{}
