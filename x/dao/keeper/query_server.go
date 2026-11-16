package keeper

import (
	"github.com/bze-alphateam/bze/x/dao/types"
)

var _ types.QueryServer = Keeper{}
