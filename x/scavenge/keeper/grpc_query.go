package keeper

import (
	"github.com/cosmonaut/bzedgev5/x/scavenge/types"
)

var _ types.QueryServer = Keeper{}
