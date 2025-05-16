package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type EpochHook struct {
	after func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error
	Name  string
}

func NewAfterEpochHook(name string, after func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error) EpochHook {
	return EpochHook{
		Name:  name,
		after: after,
	}
}

func (e EpochHook) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
	return e.after(ctx, epochIdentifier, epochNumber)
}

func (e EpochHook) BeforeEpochStart(_ sdk.Context, _ string, _ int64) error {
	return nil
}

func (e EpochHook) GetName() string {
	return e.Name
}
