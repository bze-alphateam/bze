package types_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func validFeeDest() types.FeeDestination {
	return types.FeeDestination{
		Treasury:  math.LegacyMustNewDecFromStr("0.3"),
		Burner:    math.LegacyMustNewDecFromStr("0.2"),
		Providers: math.LegacyMustNewDecFromStr("0.5"),
	}
}

func poolUpdate(id, fee string, dest types.FeeDestination) types.PoolUpdate {
	return types.PoolUpdate{PoolId: id, Fee: math.LegacyMustNewDecFromStr(fee), FeeDest: dest}
}

func TestMsgUpdatePools_ValidateBasic(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	valid := poolUpdate("stake_ubze", "0.003", validFeeDest())

	tooMany := make([]types.PoolUpdate, types.MaxUpdatesPerUpdatePoolsMsg+1)
	for i := range tooMany {
		tooMany[i] = poolUpdate(fmt.Sprintf("pool%d", i), "0.003", validFeeDest())
	}

	negative := validFeeDest()
	negative.Treasury = math.LegacyMustNewDecFromStr("-0.1")
	negative.Providers = math.LegacyMustNewDecFromStr("0.9")
	sumBelowOne := validFeeDest()
	sumBelowOne.Providers = math.LegacyMustNewDecFromStr("0.4")
	sumAboveOne := validFeeDest()
	sumAboveOne.Providers = math.LegacyMustNewDecFromStr("0.6")
	allToBurner := types.FeeDestination{Treasury: math.LegacyZeroDec(), Burner: math.LegacyOneDec(), Providers: math.LegacyZeroDec()}
	missingShare := validFeeDest()
	missingShare.Burner = math.LegacyDec{}

	tests := []struct {
		name      string
		authority string
		updates   []types.PoolUpdate
		wantErr   error
	}{
		{"valid", authority, []types.PoolUpdate{valid}, nil},
		{"max updates", authority, tooMany[:types.MaxUpdatesPerUpdatePoolsMsg], nil},
		{"fee at the minimum", authority, []types.PoolUpdate{poolUpdate("p", "0.001", validFeeDest())}, nil},
		{"fee at the maximum", authority, []types.PoolUpdate{poolUpdate("p", "0.05", validFeeDest())}, nil},
		{"whole fee to one destination", authority, []types.PoolUpdate{poolUpdate("p", "0.01", allToBurner)}, nil},
		{"invalid authority", "not-an-address", []types.PoolUpdate{valid}, nil},
		{"no updates", authority, nil, sdkerrors.ErrInvalidRequest},
		{"too many", authority, tooMany, sdkerrors.ErrInvalidRequest},
		{"duplicate pool", authority, []types.PoolUpdate{valid, poolUpdate("stake_ubze", "0.01", validFeeDest())}, sdkerrors.ErrInvalidRequest},
		{"empty pool id", authority, []types.PoolUpdate{poolUpdate("", "0.003", validFeeDest())}, sdkerrors.ErrInvalidRequest},
		{"fee below minimum", authority, []types.PoolUpdate{poolUpdate("p", "0.0009", validFeeDest())}, sdkerrors.ErrInvalidCoins},
		{"fee above maximum", authority, []types.PoolUpdate{poolUpdate("p", "0.0501", validFeeDest())}, sdkerrors.ErrInvalidCoins},
		{"zero fee", authority, []types.PoolUpdate{poolUpdate("p", "0", validFeeDest())}, sdkerrors.ErrInvalidCoins},
		{"negative fee", authority, []types.PoolUpdate{poolUpdate("p", "-0.003", validFeeDest())}, sdkerrors.ErrInvalidCoins},
		{"missing fee", authority, []types.PoolUpdate{{PoolId: "p", FeeDest: validFeeDest()}}, sdkerrors.ErrInvalidCoins},
		{"negative share", authority, []types.PoolUpdate{poolUpdate("p", "0.003", negative)}, types.ErrNegativeFeeDestination},
		{"shares sum below one", authority, []types.PoolUpdate{poolUpdate("p", "0.003", sumBelowOne)}, types.ErrInvalidFeeDestination},
		{"shares sum above one", authority, []types.PoolUpdate{poolUpdate("p", "0.003", sumAboveOne)}, types.ErrInvalidFeeDestination},
		{"missing share", authority, []types.PoolUpdate{poolUpdate("p", "0.003", missingShare)}, types.ErrInvalidFeeDestination},
		{"one invalid entry fails the batch", authority, []types.PoolUpdate{valid, poolUpdate("p", "0.5", validFeeDest())}, sdkerrors.ErrInvalidCoins},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := types.NewMsgUpdatePools(tc.authority, tc.updates).ValidateBasic()
			switch {
			case tc.name == "invalid authority":
				require.Error(t, err)
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePoolFee(t *testing.T) {
	require.NoError(t, types.ValidatePoolFee(types.MinPoolFee))
	require.NoError(t, types.ValidatePoolFee(types.MaxPoolFee))
	require.ErrorIs(t, types.ValidatePoolFee(math.LegacyZeroDec()), sdkerrors.ErrInvalidCoins)
	require.ErrorIs(t, types.ValidatePoolFee(types.MinPoolFee.Sub(math.LegacySmallestDec())), sdkerrors.ErrInvalidCoins)
	require.ErrorIs(t, types.ValidatePoolFee(types.MaxPoolFee.Add(math.LegacySmallestDec())), sdkerrors.ErrInvalidCoins)
}

func TestValidatePoolFeeDestination(t *testing.T) {
	require.NoError(t, types.ValidatePoolFeeDestination(validFeeDest()))

	negative := validFeeDest()
	negative.Burner = math.LegacyMustNewDecFromStr("-0.2")
	negative.Providers = math.LegacyMustNewDecFromStr("0.9")
	require.ErrorIs(t, types.ValidatePoolFeeDestination(negative), types.ErrNegativeFeeDestination)

	off := validFeeDest()
	off.Treasury = math.LegacyMustNewDecFromStr("0.31")
	require.ErrorIs(t, types.ValidatePoolFeeDestination(off), types.ErrInvalidFeeDestination)
}

// The proposal JSON documented in x/tradebin/docs/README.md decodes through the codec the way
// `bzed tx gov submit-proposal` reads its messages: the fee and the shares are plain decimals.
func TestMsgUpdatePools_ProposalJSONDecodes(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	raw := `{
      "@type": "/bze.tradebin.MsgUpdatePools",
      "authority": "` + authority + `",
      "updates": [
        {
          "pool_id": "stake_ubze",
          "fee": "0.003",
          "fee_dest": { "treasury": "0.25", "burner": "0.25", "providers": "0.5" }
        }
      ]
    }`

	var msg sdk.Msg
	require.NoError(t, cdc.UnmarshalInterfaceJSON([]byte(raw), &msg))
	upd, ok := msg.(*types.MsgUpdatePools)
	require.True(t, ok)
	require.NoError(t, upd.ValidateBasic())
	require.Len(t, upd.Updates, 1)
	require.True(t, upd.Updates[0].Fee.Equal(math.LegacyMustNewDecFromStr("0.003")))
	require.True(t, upd.Updates[0].FeeDest.Treasury.Equal(math.LegacyMustNewDecFromStr("0.25")))
	require.True(t, upd.Updates[0].FeeDest.Burner.Equal(math.LegacyMustNewDecFromStr("0.25")))
	require.True(t, upd.Updates[0].FeeDest.Providers.Equal(math.LegacyMustNewDecFromStr("0.5")))

	// the encoded tx carries the same values
	bz, err := cdc.MarshalInterfaceJSON(upd)
	require.NoError(t, err)
	var again sdk.Msg
	require.NoError(t, cdc.UnmarshalInterfaceJSON(bz, &again))
	require.Equal(t, upd, again)
}
