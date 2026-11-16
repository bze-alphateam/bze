package helpers

import (
	"encoding/json"
	"fmt"
)

// TradebinDecParams are the tradebin params of type LegacyDec.
var TradebinDecParams = []string{"feeMaxPriceDrift", "moduleMaxPriceDrift"}

// DecParamsForTx rewrites the named LegacyDec fields of a params query result
// into the decimal form ("0.100000000000000000") that a proposal file or a
// signed JSON tx decodes, so the map can be sent back in a MsgUpdateParams.
// The query prints a field either as a decimal (cosmos.Dec scalar) or as its
// raw 18-decimal integer (plain custom type); both are read by ParseDec. A
// field the params do not have (an older release) is left out.
func DecParamsForTx(params map[string]json.RawMessage, fields ...string) error {
	for _, name := range fields {
		raw, ok := params[name]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("param %s: %w", name, err)
		}
		d, err := ParseDec(s)
		if err != nil {
			return fmt.Errorf("param %s: %w", name, err)
		}
		out, err := json.Marshal(d.String())
		if err != nil {
			return fmt.Errorf("param %s: %w", name, err)
		}
		params[name] = out
	}
	return nil
}
