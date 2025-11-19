package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgAcceptDomain(t *testing.T) {
	authority := sample.AccAddress()
	domain := "example.com"
	active := true

	msg := NewMsgAcceptDomain(authority, domain, active)

	require.Equal(t, authority, msg.Authority)
	require.Equal(t, domain, msg.Domain)
	require.Equal(t, active, msg.Active)
}

func TestMsgAcceptDomain_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgAcceptDomain
		err  error
	}{
		{
			name: "invalid authority address",
			msg: MsgAcceptDomain{
				Authority: "invalid_address",
				Domain:    "example.com",
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty authority address",
			msg: MsgAcceptDomain{
				Authority: "",
				Domain:    "example.com",
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid domain - empty",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - no extension",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "valid domain - starts with number and letter",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "1aexample.com",
				Active:    true,
			},
		},
		{
			name: "invalid domain - contains spaces",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "exam ple.com",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - invalid characters",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example$.com",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - extension too long",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example.verylongextension",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - starts with dot",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    ".example.com",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - ends with dot",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example.com.",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "invalid domain - double dots",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example..com",
				Active:    true,
			},
			err: ErrInvalidProposalContent,
		},
		{
			name: "valid domain - simple",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - subdomain",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "sub.example.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - with numbers",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example123.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - with hyphen",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "my-example.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - with underscore",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "my_example.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - starts with letter and number",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "a1example.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - starts with number and letter",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "1aexample.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - single letter",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "a.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - two letters",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "ab.com",
				Active:    true,
			},
		},
		{
			name: "valid domain - long extension",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example.photography",
				Active:    false,
			},
		},
		{
			name: "valid domain - country code",
			msg: MsgAcceptDomain{
				Authority: sample.AccAddress(),
				Domain:    "example.co.uk",
				Active:    false,
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
