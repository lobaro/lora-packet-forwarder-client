// Package band provides band specific defaults and configuration.
package band

import (
	"errors"
	"fmt"
	"time"
)

// Name defines the band-name type.
type Name string


// Modulation defines the modulation type.
type Modulation string

// Possible modulation types.
const (
	LoRaModulation Modulation = "LORA"
	FSKModulation  Modulation = "FSK"
)

// DataRate defines a data rate
type DataRate struct {
	Modulation   Modulation `json:"modulation"`
	SpreadFactor int        `json:"spreadFactor,omitempty"` // used for LoRa
	Bandwidth    int        `json:"bandwidth,omitempty"`    // in kHz, used for LoRa
	BitRate      int        `json:"bitRate,omitempty"`      // bits per second, used for FSK
}

// MaxPayloadSize defines the max payload size
type MaxPayloadSize struct {
	M int // The maximum MACPayload size length
	N int // The maximum application payload length in the absence of the optional FOpt control field
}

// Channel defines the channel structure
type Channel struct {
	Frequency int   // frequency in Hz
	DataRates []int // each int mapping to an index in DataRateConfiguration
}

// Band defines an region specific ISM band implementation for LoRa.
type Band struct {
	// DefaultTXPower defines the default radiated transmit output power
	DefaultTXPower int

	// ImplementsCFlist defines if the band implements the optional channel
	// frequency list.
	ImplementsCFlist bool

	// RX2Frequency defines the fixed frequency for the RX2 receive window
	RX2Frequency int

	// RX2DataRate defines the fixed data-rate for the RX2 receive window
	RX2DataRate int

	// MaxFcntGap defines the MAC_FCNT_GAP default value.
	MaxFCntGap uint32

	// ADRACKLimit defines the ADR_ACK_LIMIT default value.
	ADRACKLimit int

	// ADRACKDelay defines the ADR_ACK_DELAY default value.
	ADRACKDelay int

	// ReceiveDelay1 defines the RECEIVE_DELAY1 default value.
	ReceiveDelay1 time.Duration

	// ReceiveDelay2 defines the RECEIVE_DELAY2 default value.
	ReceiveDelay2 time.Duration

	// JoinAcceptDelay1 defines the JOIN_ACCEPT_DELAY1 default value.
	JoinAcceptDelay1 time.Duration

	// JoinAcceptDelay2 defines the JOIN_ACCEPT_DELAY2 default value.
	JoinAcceptDelay2 time.Duration

	// ACKTimeoutMin defines the ACK_TIMEOUT min. default value.
	ACKTimeoutMin time.Duration

	// ACKTimeoutMax defines the ACK_TIMEOUT max. default value.
	ACKTimeoutMax time.Duration

	// DataRates defines the available data rates.
	DataRates []DataRate

	// MaxPayloadSize defines the maximum payload size, per data-rate.
	MaxPayloadSize []MaxPayloadSize

	// RX1DataRate defines the RX1 data-rate given the uplink data-rate
	// and a RX1DROffset value.
	RX1DataRate [][]int

	// TXPower defines the TX power configuration.
	TXPower []int

	// UplinkChannels defines the list of (default) configured uplink channels.
	UplinkChannels []Channel

	// DownlinkChannels defines the list of (default) configured downlink
	// channels.
	DownlinkChannels []Channel

	// getRX1ChannelFunc implements a function which returns the RX1 channel
	// based on the uplink / TX channel.
	getRX1ChannelFunc func(txChannel int) int

	// getRX1FrequencyFunc implements a function which returns the RX1 frequency
	// given the uplink frequency.
	getRX1FrequencyFunc func(band *Band, txFrequency int) (int, error)
}

// GetRX1Channel returns the channel to use for RX1 given the channel used
// for uplink.
func (b *Band) GetRX1Channel(txChannel int) int {
	return b.getRX1ChannelFunc(txChannel)
}

// GetRX1Frequency returns the frequency to use for RX1 given the uplink
// frequency.
func (b *Band) GetRX1Frequency(txFrequency int) (int, error) {
	return b.getRX1FrequencyFunc(b, txFrequency)
}


// GetDataRate returns the index of the given DataRate.
func (b *Band) GetDataRate(dr DataRate) (int, error) {
	for i, d := range b.DataRates {
		if d == dr {
			return i, nil
		}
	}
	return 0, errors.New("lorawan/band: the given data-rate does not exist")
}

// GetRX1DataRateForOffset returns the data-rate for the given offset.
func (b *Band) GetRX1DataRateForOffset(dr, drOffset int) (int, error) {
	if dr >= len(b.RX1DataRate) {
		return 0, fmt.Errorf("lorawan/band: invalid data-rate: %d", dr)
	}

	if drOffset >= len(b.RX1DataRate[dr]) {
		return 0, fmt.Errorf("lorawan/band: invalid data-rate offset: %d", drOffset)
	}
	return b.RX1DataRate[dr][drOffset], nil
}

