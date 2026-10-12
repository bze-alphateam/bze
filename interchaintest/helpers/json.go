package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Int64 decodes an integer that the CLI prints either as a JSON number or as a
// string (protobuf JSON prints 64-bit integers as strings).
type Int64 int64

// UnmarshalJSON implements json.Unmarshaler.
func (i *Int64) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if s == "" {
			*i = 0
			return nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int64 %q: %w", s, err)
		}
		*i = Int64(v)
		return nil
	}
	var v int64
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("parse int64 %s: %w", b, err)
	}
	*i = Int64(v)
	return nil
}
