package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func TestMsgSetDenomConfig_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	cp := sample.AccAddress()
	denom := "factory/" + creator + "/token"
	zero := math.LegacyZeroDec()
	fee := math.LegacyMustNewDecFromStr("0.001")
	tests := []struct {
		name string
		msg  *types.MsgSetDenomConfig
		err  error
		// fails is set when the error is not a registered sentinel (sdk.ValidateDenom errors)
		fails bool
	}{
		{name: "valid full config", msg: types.NewMsgSetDenomConfig(creator, denom, cp, fee, math.LegacyOneDec())},
		{name: "valid defaults", msg: types.NewMsgSetDenomConfig(creator, denom, "", zero, zero)},
		{name: "valid at the caps", msg: types.NewMsgSetDenomConfig(creator, denom, cp, types.MaxTransferFee, types.MaxGasSurcharge)},
		{name: "invalid creator", msg: types.NewMsgSetDenomConfig("invalid_address", denom, cp, fee, zero), err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: types.NewMsgSetDenomConfig(creator, "", cp, fee, zero), fails: true},
		{name: "native denom", msg: types.NewMsgSetDenomConfig(creator, "ubze", cp, fee, zero), err: types.ErrInvalidDenom},
		{name: "invalid community pool", msg: types.NewMsgSetDenomConfig(creator, denom, "bze1invalid", fee, zero), err: types.ErrInvalidDenomConfig},
		{name: "missing transfer fee", msg: types.NewMsgSetDenomConfig(creator, denom, cp, math.LegacyDec{}, zero), err: types.ErrInvalidDenomConfig},
		{name: "missing gas surcharge", msg: types.NewMsgSetDenomConfig(creator, denom, cp, fee, math.LegacyDec{}), err: types.ErrInvalidDenomConfig},
		{name: "negative transfer fee", msg: types.NewMsgSetDenomConfig(creator, denom, cp, fee.Neg(), zero), err: types.ErrInvalidDenomConfig},
		{name: "transfer fee above the cap", msg: types.NewMsgSetDenomConfig(creator, denom, cp, math.LegacyMustNewDecFromStr("0.06"), zero), err: types.ErrInvalidDenomConfig},
		{name: "gas surcharge above the cap", msg: types.NewMsgSetDenomConfig(creator, denom, cp, fee, math.LegacyMustNewDecFromStr("2.5")), err: types.ErrInvalidDenomConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.fails {
				require.Error(t, err)
				return
			}
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// A rate left out of the encoded message decodes as a nil decimal and is refused, not read as zero.
func TestMsgSetDenomConfig_MissingRateOnTheWire(t *testing.T) {
	creator := sample.AccAddress()
	sent := types.MsgSetDenomConfig{Creator: creator, Denom: "factory/" + creator + "/token", TransferFee: math.LegacyZeroDec(), GasSurcharge: math.LegacyZeroDec()}
	bz, err := sent.Marshal()
	require.NoError(t, err)

	// drop the gas_surcharge field (5): protobuf field tag 0x2a, then length and the bytes of "0"
	trimmed := bz[:len(bz)-3]
	require.Equal(t, []byte{0x2a, 0x01, '0'}, bz[len(bz)-3:])

	var got types.MsgSetDenomConfig
	require.NoError(t, got.Unmarshal(trimmed))
	require.True(t, got.GasSurcharge.IsNil())
	require.ErrorIs(t, got.ValidateBasic(), types.ErrInvalidDenomConfig)
}
