package gateway

import (
	"bytes"
	"testing"
)

func TestMac_MarshalText(t *testing.T) {
	m := Mac{1, 2, 3, 4, 5, 6, 7, 8}

	text, err := m.MarshalText()

	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(text, []byte("0102030405060708")) {
		t.Errorf("Expected text to be 12345678 but was %s", string(text))
	}
}

func TestMac_UnmarshalText(t *testing.T) {
	m := Mac{}

	err := m.UnmarshalText([]byte("0102030405060708"))

	if err != nil {
		t.Error(err)
	}

	if m.String() != "0102030405060708" {
		t.Errorf("Expected mac to be 0102030405060708 but was %s", m.String())
	}
}
