package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

var _ types.QueryServer = Keeper{}
