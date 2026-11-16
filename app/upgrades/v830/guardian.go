package v830

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
)

// GuardianAddress is the initial guardian of the IBC rate limits: the project
// lead's address. The guardian can only lower a quota; a gov MsgUpdateParams of
// x/guardian changes or clears the address later.
const GuardianAddress = "bze1jx4x3kn8mlz2s03zdpf2a38x9gl66llvjqdd55"

// GuardianKeeper is the part of the x/guardian keeper the handler uses.
type GuardianKeeper interface {
	SetParams(ctx context.Context, params guardiantypes.Params) error
}

// setGuardian stores the initial guardian. It runs after RunMigrations, which
// initialised x/guardian with its default genesis (no guardian).
func setGuardian(ctx sdk.Context, k GuardianKeeper) error {
	return k.SetParams(ctx, guardiantypes.NewParams(GuardianAddress))
}
