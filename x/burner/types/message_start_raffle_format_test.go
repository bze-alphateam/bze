package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/stretchr/testify/require"
)

// nonCanonicalSmallInts are integer strings that are not in canonical base-10 form.
var nonCanonicalSmallInts = []string{"0100", "010", "00", "+10", "0x10", "0o10", "0b10", "1_0", "-0", "", " ", " 10", "10 "}

// nonCanonicalPots are like nonCanonicalSmallInts but at or above the pot minimum, so only the form is wrong.
var nonCanonicalPots = []string{"0400000", "0100000", "00", "+100000", "0x186a0", "0o303240", "0b11000011010100000", "100_000", "-0", "", " ", " 100000", "100000 "}

func validStartRaffle() *types.MsgStartRaffle {
	return types.NewMsgStartRaffle(sample.AccAddress(), "100000", "30", "100", "0.5", "10", "utoken")
}

func TestAmountFormat_StartRaffleRejectsNonCanonical(t *testing.T) {
	fields := []struct {
		name   string
		values []string
		set    func(*types.MsgStartRaffle, string)
	}{
		{"pot", nonCanonicalPots, func(m *types.MsgStartRaffle, v string) { m.Pot = v }},
		{"ticket_price", nonCanonicalSmallInts, func(m *types.MsgStartRaffle, v string) { m.TicketPrice = v }},
		{"duration", nonCanonicalSmallInts, func(m *types.MsgStartRaffle, v string) { m.Duration = v }},
		{"chances", nonCanonicalSmallInts, func(m *types.MsgStartRaffle, v string) { m.Chances = v }},
	}

	for _, f := range fields {
		for _, v := range f.values {
			t.Run(f.name+"/"+v, func(t *testing.T) {
				msg := validStartRaffle()
				f.set(msg, v)
				require.Error(t, msg.ValidateBasic(), "%s %q must be refused", f.name, v)
				_, err := msg.ToStorageRaffle()
				require.Error(t, err, "%s %q must be refused", f.name, v)
			})
		}
	}
}

func TestAmountFormat_StartRaffleAcceptsCanonical(t *testing.T) {
	msg := validStartRaffle()
	msg.Pot = "123456789"
	msg.TicketPrice = "1"
	msg.Duration = "180"
	msg.Chances = "1000000"
	require.NoError(t, msg.ValidateBasic())

	raffle, err := msg.ToStorageRaffle()
	require.NoError(t, err)
	require.Equal(t, "123456789", raffle.Pot)
	require.Equal(t, "1", raffle.TicketPrice)
	require.Equal(t, uint64(180), raffle.Duration)
	require.Equal(t, uint64(1000000), raffle.Chances)
}

// The ratio keeps its accepted decimal forms: clients send it with two decimals.
func TestAmountFormat_StartRaffleRatioKeepsDecimalForms(t *testing.T) {
	for _, ratio := range []string{"0.01", "0.25", "0.5", "1", "1.00", "0.250000000000000000"} {
		t.Run(ratio, func(t *testing.T) {
			msg := validStartRaffle()
			msg.Ratio = ratio
			require.NoError(t, msg.ValidateBasic())
		})
	}
}
