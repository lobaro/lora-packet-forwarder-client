package gateway

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/Lobaro/lora-packet-forwarder-client/gateway/band"
	log "github.com/Sirupsen/logrus"
)

var errGatewayDoesNotExist = errors.New("gateway does not exist")
var gatewayCleanupDuration = -1 * time.Minute
var loRaDataRateRegex = regexp.MustCompile(`SF(\d+)BW(\d+)`)

type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

type gateway struct {
	addr            *net.UDPAddr
	lastSeen        time.Time
	protocolVersion uint8
}

type gateways struct {
	sync.RWMutex
	gateways map[Mac]gateway
	onNew    func(Mac) error
	onDelete func(Mac) error
}

func (c *gateways) get(mac Mac) (gateway, error) {
	defer c.RUnlock()
	c.RLock()
	gw, ok := c.gateways[mac]
	if !ok {
		return gw, errGatewayDoesNotExist
	}
	return gw, nil
}

func (c *gateways) set(mac Mac, gw gateway) error {
	defer c.Unlock()
	c.Lock()
	_, ok := c.gateways[mac]
	if !ok && c.onNew != nil {
		if err := c.onNew(mac); err != nil {
			return err
		}
	}
	c.gateways[mac] = gw
	return nil
}

func (c *gateways) cleanup() error {
	defer c.Unlock()
	c.Lock()
	for mac := range c.gateways {
		if c.gateways[mac].lastSeen.Before(time.Now().Add(gatewayCleanupDuration)) {
			if c.onDelete != nil {
				if err := c.onDelete(mac); err != nil {
					return err
				}
			}
			delete(c.gateways, mac)
		}
	}
	return nil
}

// Client implements a Semtech gateway client/backend.
type Client struct {
	CheckCrc bool

	log         *log.Logger
	conn        *net.UDPConn
	rxChan      chan RXPacketBytes
	statsChan   chan GatewayStatsPacket
	udpSendChan chan udpPacket
	closed      bool
	gateways    gateways
	wg          sync.WaitGroup
}

// NewClient creates a new Client.
func NewClient(bind string, onNew func(Mac) error, onDelete func(Mac) error) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return nil, err
	}
	log.WithField("addr", addr).Info("gateway: starting gateway udp listener")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	c := &Client{
		CheckCrc:    true,
		conn:        conn,
		rxChan:      make(chan RXPacketBytes),
		statsChan:   make(chan GatewayStatsPacket),
		udpSendChan: make(chan udpPacket),
		gateways: gateways{
			gateways: make(map[Mac]gateway),
			onNew:    onNew,
			onDelete: onDelete,
		},
	}

	go func() {
		for {
			if err := c.gateways.cleanup(); err != nil {
				c.log.Errorf("gateway: gateways cleanup failed: %s", err)
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		c.wg.Add(1)
		err := c.readPackets()
		if !c.closed {
			c.log.Fatal(err)
		}
		c.wg.Done()
	}()

	go func() {
		c.wg.Add(1)
		err := c.sendPackets()
		if !c.closed {
			c.log.Fatal(err)
		}
		c.wg.Done()
	}()

	return c, nil
}

func (c *Client) SetLogger(logger *log.Logger) {
	c.log = logger
}

// Close closes the client.
func (c *Client) Close() error {
	c.log.Info("gateway: closing gateway client")
	c.closed = true
	close(c.udpSendChan)
	if err := c.conn.Close(); err != nil {
		return err
	}
	c.log.Info("gateway: handling last packets")
	c.wg.Wait()
	return nil
}

// RXPacketChan returns the channel containing the received RX packets.
func (c *Client) RXPacketChan() chan RXPacketBytes {
	return c.rxChan
}

// StatsChan returns the channel containg the received gateway stats.
func (c *Client) StatsChan() chan GatewayStatsPacket {
	return c.statsChan
}

// Send sends the given packet to the gateway.
func (c *Client) Send(txPacket TXPacketBytes) error {
	gw, err := c.gateways.get(txPacket.TXInfo.MAC)
	if err != nil {
		return err
	}
	txpk, err := newTXPKFromTXPacket(txPacket)
	if err != nil {
		return err
	}
	pullResp := PullRespPacket{
		ProtocolVersion: gw.protocolVersion,
		Payload: PullRespPayload{
			TXPK: txpk,
		},
	}
	bytes, err := pullResp.MarshalBinary()
	if err != nil {
		return fmt.Errorf("gateway: json marshall PullRespPacket error: %s", err)
	}
	c.udpSendChan <- udpPacket{
		data: bytes,
		addr: gw.addr,
	}
	return nil
}

func (c *Client) readPackets() error {
	buf := make([]byte, 65507) // max udp data size
	for {
		i, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("gateway: read from udp error: %s", err)
		}
		data := make([]byte, i)
		copy(data, buf[:i])
		go func(data []byte) {
			if err := c.handlePacket(addr, data); err != nil {
				c.log.WithFields(log.Fields{
					"data_base64": base64.StdEncoding.EncodeToString(data),
					"addr":        addr,
				}).Errorf("gateway: could not handle packet: %s", err)
			}
		}(data)
	}
}

func (c *Client) sendPackets() error {
	for p := range c.udpSendChan {
		pt, err := GetPacketType(p.data)
		if err != nil {
			c.log.WithFields(log.Fields{
				"addr":        p.addr,
				"data_base64": base64.StdEncoding.EncodeToString(p.data),
			}).Error("gateway: unknown packet type")
			continue
		}
		c.log.WithFields(log.Fields{
			"addr":             p.addr,
			"type":             pt,
			"protocol_version": p.data[0],
		}).Info("gateway: sending udp packet to gateway")

		if _, err := c.conn.WriteToUDP(p.data, p.addr); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) handlePacket(addr *net.UDPAddr, data []byte) error {
	pt, err := GetPacketType(data)
	if err != nil {
		return err
	}
	c.log.WithFields(log.Fields{
		"addr":             addr,
		"type":             pt,
		"protocol_version": data[0],
	}).Info("gateway: received udp packet from gateway")

	switch pt {
	case PushData:
		return c.handlePushData(addr, data)
	case PullData:
		return c.handlePullData(addr, data)
	case TXACK:
		return c.handleTXACK(addr, data)
	default:
		return fmt.Errorf("gateway: unknown packet type: %s", pt)
	}
}

func (b *Client) handlePullData(addr *net.UDPAddr, data []byte) error {
	var p PullDataPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}
	ack := PullACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return err
	}

	err = b.gateways.set(p.GatewayMAC, gateway{
		addr:            addr,
		lastSeen:        time.Now().UTC(),
		protocolVersion: p.ProtocolVersion,
	})
	if err != nil {
		return err
	}

	b.udpSendChan <- udpPacket{
		addr: addr,
		data: bytes,
	}
	return nil
}

func (b *Client) handlePushData(addr *net.UDPAddr, data []byte) error {
	var p PushDataPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}

	// ack the packet
	ack := PushACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return err
	}
	b.udpSendChan <- udpPacket{
		addr: addr,
		data: bytes,
	}

	// gateway stats
	if p.Payload.Stat != nil {
		b.handleStat(addr, p.GatewayMAC, *p.Payload.Stat)
	}

	// rx packets
	for _, rxpk := range p.Payload.RXPK {
		if err := b.handleRXPacket(addr, p.GatewayMAC, rxpk); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) handleStat(addr *net.UDPAddr, mac Mac, stat Stat) {
	gwStats := newGatewayStatsPacket(mac, stat)
	c.log.WithFields(log.Fields{
		"addr": addr,
		"mac":  mac,
	}).Info("gateway: stat packet received")
	addIPToGatewayStatsPacket(&gwStats, addr.IP)
	if gtw, err := c.gateways.get(mac); err != nil && gtw.addr != nil {
		addIPToGatewayStatsPacket(&gwStats, gtw.addr.IP)
	}
	c.statsChan <- gwStats
}

func (c *Client) handleRXPacket(addr *net.UDPAddr, mac Mac, rxpk RXPK) error {
	logFields := log.Fields{
		"addr": addr,
		"mac":  mac,
		"data": rxpk.Data,
	}
	c.log.WithFields(logFields).Info("gateway: rxpk packet received")

	// decode packet
	rxPacket, err := newRXPacketFromRXPK(mac, rxpk)
	if err != nil {
		return err
	}

	// check CRC
	if c.CheckCrc && rxPacket.RXInfo.CRCStatus != 1 {
		c.log.WithFields(logFields).Warningf("gateway: invalid packet CRC: %d", rxPacket.RXInfo.CRCStatus)
		return errors.New("gateway: invalid CRC")
	}
	c.rxChan <- rxPacket
	return nil
}

func (c *Client) handleTXACK(addr *net.UDPAddr, data []byte) error {
	var p TXACKPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}
	var errBool bool

	logFields := log.Fields{
		"mac":          p.GatewayMAC,
		"random_token": p.RandomToken,
	}
	if p.Payload != nil {
		if p.Payload.TXPKACK.Error != "NONE" {
			errBool = true
		}
		logFields["error"] = p.Payload.TXPKACK.Error
	}

	if errBool {
		c.log.WithFields(logFields).Error("gateway: tx ack received")
	} else {
		c.log.WithFields(logFields).Info("gateway: tx ack received")
	}

	return nil
}

// newGatewayStatsPacket from Stat transforms a Semtech Stat packet into a
// GatewayStatsPacket.
func newGatewayStatsPacket(mac Mac, stat Stat) GatewayStatsPacket {
	return GatewayStatsPacket{
		Time:                time.Time(stat.Time),
		MAC:                 mac,
		Latitude:            stat.Lati,
		Longitude:           stat.Long,
		Altitude:            float64(stat.Alti),
		RXPacketsReceived:   int(stat.RXNb),
		RXPacketsReceivedOK: int(stat.RXOK),
		CustomData: map[string]interface{}{
			"platform":     stat.Pfrm,
			"contactEmail": stat.Mail,
			"description":  stat.Desc,
			"ip":           []string{},
		},
	}
}

// newRXPacketFromRXPK transforms a Semtech packet into a RXPacketBytes.
func newRXPacketFromRXPK(mac Mac, rxpk RXPK) (RXPacketBytes, error) {
	dataRate, err := newDataRateFromDatR(rxpk.DatR)
	if err != nil {
		return RXPacketBytes{}, fmt.Errorf("gateway: could not get DataRate from DatR: %s", err)
	}

	b, err := base64.StdEncoding.DecodeString(rxpk.Data)
	if err != nil {
		return RXPacketBytes{}, fmt.Errorf("gateway: could not base64 decode data: %s", err)
	}

	rxPacket := RXPacketBytes{
		PHYPayload: b,
		RXInfo: RXInfo{
			MAC:       mac,
			Time:      time.Time(rxpk.Time),
			Timestamp: rxpk.Tmst,
			Frequency: int(rxpk.Freq * 1000000),
			Channel:   int(rxpk.Chan),
			RFChain:   int(rxpk.RFCh),
			CRCStatus: int(rxpk.Stat),
			DataRate:  dataRate,
			CodeRate:  rxpk.CodR,
			RSSI:      int(rxpk.RSSI),
			LoRaSNR:   rxpk.LSNR,
			Size:      int(rxpk.Size),
		},
	}
	return rxPacket, nil
}

// newTXPKFromTXPacket transforms a TXPacketBytes into a Semtech
// compatible packet.
func newTXPKFromTXPacket(txPacket TXPacketBytes) (TXPK, error) {
	txpk := TXPK{
		Imme: txPacket.TXInfo.Immediately,
		Tmst: txPacket.TXInfo.Timestamp,
		Freq: float64(txPacket.TXInfo.Frequency) / 1000000,
		Powe: uint8(txPacket.TXInfo.Power),
		Modu: string(txPacket.TXInfo.DataRate.Modulation),
		DatR: newDatRfromDataRate(txPacket.TXInfo.DataRate),
		CodR: txPacket.TXInfo.CodeRate,
		Size: uint16(len(txPacket.PHYPayload)),
		Data: base64.StdEncoding.EncodeToString(txPacket.PHYPayload),
	}

	if txPacket.TXInfo.DataRate.Modulation == band.FSKModulation {
		txpk.FDev = uint16(txPacket.TXInfo.DataRate.BitRate / 2)
	}

	// by default IPol=true is used for downlink LoRa modulation, however in
	// some cases one might want to override this.
	if txPacket.TXInfo.IPol != nil {
		txpk.IPol = *txPacket.TXInfo.IPol
	} else if txPacket.TXInfo.DataRate.Modulation == band.LoRaModulation {
		txpk.IPol = true
	}

	return txpk, nil
}

func newDataRateFromDatR(d DatR) (band.DataRate, error) {
	var dr band.DataRate

	if d.LoRa != "" {
		// parse e.g. SF12BW250 into separate variables
		match := loRaDataRateRegex.FindStringSubmatch(d.LoRa)
		if len(match) != 3 {
			return dr, errors.New("gateway: could not parse LoRa data rate")
		}

		// cast variables to ints
		sf, err := strconv.Atoi(match[1])
		if err != nil {
			return dr, fmt.Errorf("gateway: could not convert spread factor to int: %s", err)
		}
		bw, err := strconv.Atoi(match[2])
		if err != nil {
			return dr, fmt.Errorf("gateway: could not convert bandwith to int: %s", err)
		}

		dr.Modulation = band.LoRaModulation
		dr.SpreadFactor = sf
		dr.Bandwidth = bw
		return dr, nil
	}

	if d.FSK != 0 {
		dr.Modulation = band.FSKModulation
		dr.BitRate = int(d.FSK)
		return dr, nil
	}

	return dr, errors.New("gateway: could not convert DatR to DataRate, DatR is empty / modulation unknown")
}

func newDatRfromDataRate(d band.DataRate) DatR {
	if d.Modulation == band.LoRaModulation {
		return DatR{
			LoRa: fmt.Sprintf("SF%dBW%d", d.SpreadFactor, d.Bandwidth),
		}
	}

	return DatR{
		FSK: uint32(d.BitRate),
	}
}
