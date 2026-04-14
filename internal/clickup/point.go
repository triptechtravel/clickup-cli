package clickup

import (
	"encoding/json"
)

// Point represents a sprint/story points value with precision-preserving JSON handling.
type Point struct {
	Value json.Number

	IntVal   *int64
	FloatVal *float64
}

// MarshalJSON implements json.Marshaler using a value receiver so that
// encoding/json calls it on non-pointer struct fields.
func (p Point) MarshalJSON() ([]byte, error) {
	if p.IntVal != nil {
		return json.Marshal(p.IntVal)
	}

	if p.FloatVal != nil {
		return json.Marshal(p.FloatVal)
	}

	return json.Marshal(p.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Point) UnmarshalJSON(b []byte) error {
	p.IntVal = nil
	p.FloatVal = nil

	var i int64
	var f float64
	if err := json.Unmarshal(b, &i); err == nil {
		p.IntVal = &i
	} else {
		if err = json.Unmarshal(b, &f); err == nil {
			p.FloatVal = &f
		} else {
			return err
		}
	}

	p.Value = json.Number(b)
	return nil
}
