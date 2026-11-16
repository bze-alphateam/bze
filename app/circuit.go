package app

import (
	"context"

	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"

	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
)

// circuitDisableList gives x/guardian the x/circuit disable list. It can read
// the list and add to it, never remove from it: only governance re-enables a
// message, with x/circuit's MsgResetCircuitBreaker.
type circuitDisableList struct {
	keeper *circuitkeeper.Keeper
}

var _ guardiantypes.CircuitKeeper = circuitDisableList{}

func (c circuitDisableList) IsDisabled(ctx context.Context, msgTypeURL string) (bool, error) {
	allowed, err := c.keeper.IsAllowed(ctx, msgTypeURL)
	return !allowed, err
}

func (c circuitDisableList) Disable(ctx context.Context, msgTypeURL string) error {
	return c.keeper.DisableList.Set(ctx, msgTypeURL)
}

// msgRouter tells x/guardian whether a type URL is routed to a Msg service.
type msgRouter struct {
	router *baseapp.MsgServiceRouter
}

var _ guardiantypes.MsgRouter = msgRouter{}

func (r msgRouter) IsRegistered(msgTypeURL string) bool {
	return r.router.HandlerByTypeURL(msgTypeURL) != nil
}

// wireGuardianCircuit hands x/guardian the circuit disable list and the
// message router, which exist only once the app is built.
//
// The disable list is enforced by x/circuit itself: its depinject provider
// returns a baseapp option that sets the keeper as the router's circuit
// breaker, so the router refuses a disabled type whether the message comes in
// a tx or is executed by another module (x/gov, x/authz, x/dao), since all of
// them dispatch through the app's router.
func (app *App) wireGuardianCircuit() {
	app.GuardianKeeper.SetCircuitKeeper(circuitDisableList{keeper: &app.CircuitBreakerKeeper})
	app.GuardianKeeper.SetMsgRouter(msgRouter{router: app.MsgServiceRouter()})
}
