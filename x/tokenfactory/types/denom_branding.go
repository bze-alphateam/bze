package types

import "regexp"

var (
	brandingFontRegex  = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)
	brandingColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// IsEmpty returns true when the branding carries no data at all, in which
// case MsgSetDenomBranding is treated as a "clear branding" operation.
func (b *DenomBranding) IsEmpty() bool {
	return b == nil || (b.Font == "" && b.Light == nil && b.Dark == nil)
}

// Validate checks the branding package is complete and valid: a font slug
// plus all 8 colours of the light and dark palettes (all-or-nothing).
func (b *DenomBranding) Validate() error {
	if b == nil {
		return ErrInvalidBranding.Wrap("branding package is empty")
	}

	if !brandingFontRegex.MatchString(b.Font) {
		return ErrInvalidBranding.Wrapf("invalid font: %s", b.Font)
	}

	if err := b.Light.validate("light"); err != nil {
		return err
	}

	return b.Dark.validate("dark")
}

func (c *BrandingColors) validate(palette string) error {
	if c == nil {
		return ErrInvalidBranding.Wrapf("%s palette is missing", palette)
	}

	colors := []struct {
		name  string
		value string
	}{
		{"background", c.Background},
		{"text", c.Text},
		{"primary", c.Primary},
		{"secondary", c.Secondary},
	}
	for _, color := range colors {
		if !brandingColorRegex.MatchString(color.value) {
			return ErrInvalidBranding.Wrapf("invalid %s %s color: %s", palette, color.name, color.value)
		}
	}

	return nil
}
