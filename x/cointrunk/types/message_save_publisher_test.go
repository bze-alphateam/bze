package types

import (
	"strings"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgSavePublisher(t *testing.T) {
	authority := sample.AccAddress()
	name := "Test Publisher"
	address := sample.AccAddress()
	active := true

	msg := NewMsgSavePublisher(authority, name, address, active)

	require.Equal(t, authority, msg.Authority)
	require.Equal(t, name, msg.Name)
	require.Equal(t, address, msg.Address)
	require.Equal(t, active, msg.Active)
}

func TestMsgSavePublisher_ValidateBasic(t *testing.T) {
	validAuthority := sample.AccAddress()
	validAddress := sample.AccAddress()
	validName := "Valid Publisher Name"

	// Create strings for length testing
	shortName := "ab"                        // 2 chars, below nameMinLen (3)
	longName := strings.Repeat("a", 257)     // 257 chars, above nameMaxLen (256)
	minValidName := "abc"                    // exactly nameMinLen (3)
	maxValidName := strings.Repeat("a", 256) // exactly nameMaxLen (256)

	tests := []struct {
		name string
		msg  MsgSavePublisher
		err  error
	}{
		{
			name: "invalid authority address",
			msg: MsgSavePublisher{
				Authority: "invalid_address",
				Name:      validName,
				Address:   validAddress,
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty authority address",
			msg: MsgSavePublisher{
				Authority: "",
				Name:      validName,
				Address:   validAddress,
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "name too short",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      shortName,
				Address:   validAddress,
				Active:    true,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "empty name",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      "",
				Address:   validAddress,
				Active:    true,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "name too long",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      longName,
				Address:   validAddress,
				Active:    true,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid publisher address",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      validName,
				Address:   "invalid_address",
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty publisher address",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      validName,
				Address:   "",
				Active:    true,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "valid message - active true",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      validName,
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - active false",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      validName,
				Address:   validAddress,
				Active:    false,
			},
		},
		{
			name: "valid message - minimum name length",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      minValidName,
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - maximum name length",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      maxValidName,
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - name with spaces",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      "Publisher With Spaces",
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - name with numbers",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      "Publisher123",
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - name with special characters",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      "Publisher-Name_2024",
				Address:   validAddress,
				Active:    true,
			},
		},
		{
			name: "valid message - same authority and address",
			msg: MsgSavePublisher{
				Authority: validAuthority,
				Name:      validName,
				Address:   validAuthority, // Same as authority
				Active:    true,
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
