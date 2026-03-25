package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateLiquidityPool(t *testing.T) {
	creator := sample.AccAddress()
	base := "uatom"
	quote := "uusdc"
	fee := "0.003"
	feeDest := `{"treasury":"0.5","burner":"0.3","providers":"0.2"}`
	initialBase := math.NewInt(1000)
	initialQuote := math.NewInt(2000)

	msg := NewMsgCreateLiquidityPool(creator, base, quote, fee, feeDest, false, initialBase, initialQuote)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, base, msg.Base)
	require.Equal(t, quote, msg.Quote)
	require.Equal(t, fee, msg.Fee)
	require.Equal(t, feeDest, msg.FeeDest)
	require.Equal(t, false, msg.Stable)
	require.Equal(t, initialBase, msg.InitialBase)
	require.Equal(t, initialQuote, msg.InitialQuote)
}

func TestMsgCreateLiquidityPool_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validBase := "uatom"
	validQuote := "uusdc"
	validFee := "0.003"
	validFeeDest := `{"treasury":"0.5","burner":"0.3","providers":"0.2"}`
	validInitialBase := math.NewInt(1000)
	validInitialQuote := math.NewInt(2000)

	tests := []struct {
		name string
		msg  MsgCreateLiquidityPool
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateLiquidityPool{
				Creator:      "invalid_address",
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateLiquidityPool{
				Creator:      "",
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "missing assets - both base and quote empty",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         "",
				Quote:        "",
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "missing assets - base empty",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         "",
				Quote:        "ubze",
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "missing assets - quote empty",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         "ubze",
				Quote:        "",
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "missing fee",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          "",
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "missing fee destination",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      "",
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "negative initial base",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  math.NewInt(-1000),
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "zero initial base",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  math.NewInt(0),
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "negative initial quote",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: math.NewInt(-2000),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "zero initial quote",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: math.NewInt(0),
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid message - stable pool not supported",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       true,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - non-stable pool",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  validInitialBase,
				InitialQuote: validInitialQuote,
			},
		},
		{
			name: "valid message - large amounts",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  math.NewInt(1000000000000),
				InitialQuote: math.NewInt(2000000000000),
			},
		},
		{
			name: "valid message - minimum amounts",
			msg: MsgCreateLiquidityPool{
				Creator:      validCreator,
				Base:         validBase,
				Quote:        validQuote,
				Fee:          validFee,
				FeeDest:      validFeeDest,
				Stable:       false,
				InitialBase:  math.NewInt(1),
				InitialQuote: math.NewInt(1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgCreateLiquidityPool_ParseFeeDestination(t *testing.T) {
	msg := &MsgCreateLiquidityPool{}

	tests := []struct {
		name      string
		feeDest   string
		shouldErr bool
	}{
		{
			name:      "valid fee destination",
			feeDest:   `{"treasury":"0.5","burner":"0.3","providers":"0.2"}`,
			shouldErr: false,
		},
		{
			name:      "valid fee destination - different values",
			feeDest:   `{"treasury":"0.6","burner":"0.2","providers":"0.2"}`,
			shouldErr: false,
		},
		{
			name:      "invalid json",
			feeDest:   `{"treasury":"0.5","burner":"0.3"`,
			shouldErr: true,
		},
		{
			name:      "empty json",
			feeDest:   `{}`,
			shouldErr: false,
		},
		{
			name:      "invalid json format",
			feeDest:   `invalid_json`,
			shouldErr: true,
		},
		{
			name:      "empty string",
			feeDest:   "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg.FeeDest = tt.feeDest
			feeDest, err := msg.ParseFeeDestination()
			if tt.shouldErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, feeDest)
		})
	}
}
