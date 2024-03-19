package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type EpochHook struct {
	before func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error
	Name   string
}

func NewBeforeEpochHook(name string, before func(ctx sdk.Context, epochIdentifier string, epochNumber int64) error) EpochHook {
	return EpochHook{
		Name:   name,
		before: before,
	}
}

func (e EpochHook) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) error {
	return e.before(ctx, epochIdentifier, epochNumber)
}

func (e EpochHook) AfterEpochEnd(_ sdk.Context, _ string, _ int64) error {
	return nil
}

func (e EpochHook) GetName() string {
	return e.Name
}
