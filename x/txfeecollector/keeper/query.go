package keeper

import (
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

var _ types.QueryServer = Keeper{}
