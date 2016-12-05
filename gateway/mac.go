package gateway

import (
	"fmt"
	"encoding/hex"
	"errors"
)

// 64 bit MacAddress of Gateway
type Mac [8]byte


// MarshalText implements encoding.TextMarshaler.
func (m Mac) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *Mac) UnmarshalText(text []byte) error {
	b, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	if len(m) != len(b) {
		return fmt.Errorf("lorawan: exactly %d bytes are expected", len(m))
	}
	copy(m[:], b)
	return nil
}

// String implement fmt.Stringer.
func (m Mac) String() string {
	return hex.EncodeToString(m[:])
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (m Mac) MarshalBinary() ([]byte, error) {
	out := make([]byte, len(m))
	// little endian
	for i, v := range m {
		out[len(m)-i-1] = v
	}
	return out, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (m *Mac) UnmarshalBinary(data []byte) error {
	if len(data) != len(m) {
		return fmt.Errorf("lorawan: %d bytes of data are expected", len(m))
	}
	for i, v := range data {
		// little endian
		m[len(m)-i-1] = v
	}
	return nil
}

// Scan implements sql.Scanner.
func (m *Mac) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return errors.New("lorawan: []byte type expected")
	}
	if len(b) != len(m) {
		return fmt.Errorf("lorawan []byte must have length %d", len(m))
	}
	copy(m[:], b)
	return nil
}
