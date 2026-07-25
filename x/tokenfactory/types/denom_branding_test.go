package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func validBranding() *DenomBranding {
	return &DenomBranding{
		Font: "inter",
		Light: &BrandingColors{
			Background: "#FFFFFF",
			Text:       "#111111",
			Primary:    "#1a2b3c",
			Secondary:  "#A1b2C3",
		},
		Dark: &BrandingColors{
			Background: "#000000",
			Text:       "#eeeeee",
			Primary:    "#C0FFEE",
			Secondary:  "#abcdef",
		},
	}
}

func TestDenomBranding_Validate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(b *DenomBranding)
		expValid bool
	}{
		{
			name:     "valid package",
			mutate:   func(b *DenomBranding) {},
			expValid: true,
		},
		{
			name:     "valid font with digits and hyphens",
			mutate:   func(b *DenomBranding) { b.Font = "jetbrains-mono-2" },
			expValid: true,
		},
		{
			name:     "empty font",
			mutate:   func(b *DenomBranding) { b.Font = "" },
			expValid: false,
		},
		{
			name:     "uppercase font",
			mutate:   func(b *DenomBranding) { b.Font = "Inter" },
			expValid: false,
		},
		{
			name:     "font with invalid character",
			mutate:   func(b *DenomBranding) { b.Font = "inter_bold" },
			expValid: false,
		},
		{
			name:     "font too long",
			mutate:   func(b *DenomBranding) { b.Font = strings.Repeat("a", 33) },
			expValid: false,
		},
		{
			name:     "missing light palette",
			mutate:   func(b *DenomBranding) { b.Light = nil },
			expValid: false,
		},
		{
			name:     "missing dark palette",
			mutate:   func(b *DenomBranding) { b.Dark = nil },
			expValid: false,
		},
		{
			name:     "partial palette: empty light background",
			mutate:   func(b *DenomBranding) { b.Light.Background = "" },
			expValid: false,
		},
		{
			name:     "partial palette: empty dark secondary",
			mutate:   func(b *DenomBranding) { b.Dark.Secondary = "" },
			expValid: false,
		},
		{
			name:     "colour without hash",
			mutate:   func(b *DenomBranding) { b.Light.Text = "FFFFFF" },
			expValid: false,
		},
		{
			name:     "named colour",
			mutate:   func(b *DenomBranding) { b.Light.Primary = "red" },
			expValid: false,
		},
		{
			name:     "non-hex characters",
			mutate:   func(b *DenomBranding) { b.Dark.Background = "#GGGGGG" },
			expValid: false,
		},
		{
			name:     "short hex colour",
			mutate:   func(b *DenomBranding) { b.Dark.Text = "#FFF" },
			expValid: false,
		},
		{
			name:     "long hex colour",
			mutate:   func(b *DenomBranding) { b.Light.Secondary = "#1234567" },
			expValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branding := validBranding()
			tt.mutate(branding)
			err := branding.Validate()
			if tt.expValid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, ErrInvalidBranding)
		})
	}
}

func TestDenomBranding_Validate_Nil(t *testing.T) {
	var branding *DenomBranding
	require.ErrorIs(t, branding.Validate(), ErrInvalidBranding)
}

func TestDenomBranding_IsEmpty(t *testing.T) {
	var nilBranding *DenomBranding
	require.True(t, nilBranding.IsEmpty())
	require.True(t, (&DenomBranding{}).IsEmpty())
	require.False(t, validBranding().IsEmpty())
	require.False(t, (&DenomBranding{Font: "inter"}).IsEmpty())
	require.False(t, (&DenomBranding{Light: &BrandingColors{}}).IsEmpty())
}
