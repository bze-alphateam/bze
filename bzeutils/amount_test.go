package bzeutils_test

import (
	"testing"

	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/stretchr/testify/require"
)

func TestParseCanonicalInt(t *testing.T) {
	for _, s := range []string{"0", "1", "10", "100", "-1", "-100", "123456789012345678901234567890"} {
		t.Run(s, func(t *testing.T) {
			i, err := bzeutils.ParseCanonicalInt(s)
			require.NoError(t, err)
			require.Equal(t, s, i.String())
		})
	}

	for _, s := range []string{"0100", "00", "+10", "0x10", "0X10", "0o10", "0b10", "1_0", "-0", "-010", "", " ", " 10", "10 ", "1.0", "1e3", "abc"} {
		t.Run(s, func(t *testing.T) {
			_, err := bzeutils.ParseCanonicalInt(s)
			require.Error(t, err, "%q must be refused", s)
		})
	}
}

func TestParseCanonicalAmount(t *testing.T) {
	for _, s := range []string{"1", "100", "123456789012345678901234567890"} {
		t.Run(s, func(t *testing.T) {
			amt, err := bzeutils.ParseCanonicalAmount(s)
			require.NoError(t, err)
			require.Equal(t, s, amt.String())
		})
	}

	for _, s := range []string{"0", "-1", "0100", "+10", "0x10", ""} {
		t.Run(s, func(t *testing.T) {
			_, err := bzeutils.ParseCanonicalAmount(s)
			require.Error(t, err, "%q must be refused", s)
		})
	}
}
