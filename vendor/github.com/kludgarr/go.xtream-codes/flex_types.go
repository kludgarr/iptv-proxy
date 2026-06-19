package xtreamcodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Timestamp is a helper struct to convert unix timestamp ints and strings to time.Time.
type Timestamp struct {
	time.Time
	quoted bool
}

// MarshalJSON returns the Unix timestamp as a string.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.quoted {
		return []byte(`"` + strconv.FormatInt(t.Time.Unix(), 10) + `"`), nil
	}
	return []byte(strconv.FormatInt(t.Time.Unix(), 10)), nil
}

// UnmarshalJSON converts the int or string to a Unix timestamp.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	t.quoted = len(b) > 0 && b[0] == '"'
	s := strings.Trim(string(b), `"`)
	if len(s) == 0 {
		return nil
	}
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.Unix(ts, 0)
	return nil
}

// ConvertibleBoolean is a helper type to allow JSON documents using 0/1 or "true" and "false" be converted to bool.
type ConvertibleBoolean struct {
	bool
	quoted bool
}

// Bool returns the underlying bool value.
func (bit ConvertibleBoolean) Bool() bool { return bit.bool }

// IsZero reports whether the ConvertibleBoolean holds the zero value
// (false, unquoted). Combined with json `omitzero` (Go 1.24+) for
// targeted elision; do not apply to fields where false-vs-absent is a
// distinction the caller relies on.
func (bit ConvertibleBoolean) IsZero() bool { return !bit.bool && !bit.quoted }

// String implements fmt.Stringer.
func (bit ConvertibleBoolean) String() string {
	if bit.bool {
		return "true"
	}
	return "false"
}

// MarshalJSON returns a 0 or 1 depending on bool state.
func (bit ConvertibleBoolean) MarshalJSON() ([]byte, error) {
	var bitSetVar int8
	if bit.bool {
		bitSetVar = 1
	}

	if bit.quoted {
		return json.Marshal(fmt.Sprint(bitSetVar))
	}

	return json.Marshal(bitSetVar)
}

// UnmarshalJSON converts a 0, 1, true or false into a bool
func (bit *ConvertibleBoolean) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	bit.quoted = len(data) > 0 && data[0] == '"'
	asString := strings.Trim(string(data), `"`)
	switch asString {
	case "1", "true":
		bit.bool = true
	case "0", "false":
		bit.bool = false
	default:
		return fmt.Errorf("Boolean unmarshal error: invalid input %s", asString)
	}
	return nil
}

// FlexInt unmarshals from JSON as either a quoted or unquoted integer,
// preserving the original quoting form so round-trip marshaling is faithful.
type FlexInt struct {
	value  int64
	quoted bool
}

// NewFlexInt creates a FlexInt from an int64 value.
func NewFlexInt(v int64) FlexInt { return FlexInt{value: v} }

// IsZero reports whether the FlexInt holds the zero value. Combined with
// the json `omitzero` tag (Go 1.24+), this allows targeted elision of
// absent-equals-zero fields without affecting fields where a zero value
// carries semantic signal.
func (f FlexInt) IsZero() bool { return f.value == 0 }

// Int64 returns the underlying int64 value.
func (f FlexInt) Int64() int64 { return f.value }

// Int returns the underlying value as int.
func (f FlexInt) Int() int { return int(f.value) }

// String implements fmt.Stringer.
func (f FlexInt) String() string { return strconv.FormatInt(f.value, 10) }

func (f FlexInt) MarshalJSON() ([]byte, error) {
	if f.quoted {
		return []byte(`"` + strconv.FormatInt(f.value, 10) + `"`), nil
	}
	return json.Marshal(f.value)
}

// UnmarshalJSON is intentionally tolerant: malformed numerics coerce to
// zero rather than failing the enclosing record's decode. Providers in the
// wild emit empty strings, nulls, and occasional non-numeric strings on
// fields the contract treats as integers; record-level decode failure is
// worse than zero-coercion in this domain. Callers that need to distinguish
// "explicit zero" from "coerced bad input" should validate at a higher level.
func (f *FlexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	f.quoted = len(data) > 0 && data[0] == '"'
	data = bytes.Trim(data, `" `)
	if len(data) == 0 {
		f.value = 0
		return nil
	}
	if err := json.Unmarshal(data, &f.value); err != nil {
		f.value = 0
	}
	return nil
}

// FlexFloat unmarshals from JSON as either a quoted or unquoted float,
// preserving the original quoting form so round-trip marshaling is faithful.
type FlexFloat struct {
	value  float64
	quoted bool
}

// Float64 returns the underlying float64 value.
func (ff FlexFloat) Float64() float64 { return ff.value }

// String implements fmt.Stringer.
func (ff FlexFloat) String() string { return strconv.FormatFloat(ff.value, 'f', -1, 64) }

func (ff FlexFloat) MarshalJSON() ([]byte, error) {
	if ff.quoted {
		return []byte(`"` + strconv.FormatFloat(ff.value, 'f', -1, 64) + `"`), nil
	}
	return json.Marshal(ff.value)
}

func (ff *FlexFloat) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	ff.quoted = len(b) > 0 && b[0] == '"'
	if ff.quoted {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if len(s) == 0 {
			s = "0"
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			f = 0
		}
		ff.value = f
		return nil
	}
	return json.Unmarshal(b, &ff.value)
}

// JSONStringSlice is a struct containing a slice of strings.
// It is needed for cases in which we may get an array or may get
// a single string in a JSON response.
type JSONStringSlice struct {
	Slice        []string `json:"-"`
	SingleString bool     `json:"-"`
}

// MarshalJSON returns b as the JSON encoding of b.
func (b JSONStringSlice) MarshalJSON() ([]byte, error) {
	if !b.SingleString {
		return json.Marshal(b.Slice)
	}
	return json.Marshal(b.Slice[0])
}

// UnmarshalJSON sets *b to a copy of data.
func (b *JSONStringSlice) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		data = append([]byte(`[`), data...)
		data = append(data, []byte(`]`)...)
		b.SingleString = true
	}

	return json.Unmarshal(data, &b.Slice)
}
