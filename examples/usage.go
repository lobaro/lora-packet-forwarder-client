package examples

import (
	"github.com/Lobaro/lora-packet-forwarder-client/gateway"
	"github.com/Lobaro/lora-packet-forwarder-client/gateway/band"
	"github.com/apex/log"
)

var exit = make(chan struct{})

func Exit() {
	close(exit)
}

func main() {
	client, err := gateway.NewClient(":1680", onNewGateway, onDeleteGateway)

	if err != nil {
		panic(err)
	}

	defer client.Close()

	go handleRxPackets(client)

	log.Info("Listening on poer :1680. Press Ctrl + c to exit.")

	// Block till Exit() is called or program terminates
	<-exit
}

func onNewGateway(gwMac gateway.Mac) error {
	// New gateway with given mac address
	return nil
}

func onDeleteGateway(gwMac gateway.Mac) error {
	// Removed gateway with given mac address
	return nil
}

func handleRxPackets(client *gateway.Client) {
	for {
		select {
		case msg := <-client.RXPacketChan():
			log.Infof("Received packet from Gateway with phy payload %v", msg.PHYPayload)
		}
	}

}

// SendDownlinkExample demonstrates how to send down-link data to the gateway
func SendDownlinkExample(client *gateway.Client) {

	txPacket := gateway.TXPacketBytes{
		TXInfo: gateway.TXInfo{
			MAC:         [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Immediately: true,
			Timestamp:   12345,
			Frequency:   868100000,
			Power:       14,
			DataRate: band.DataRate{
				Modulation:   band.LoRaModulation,
				SpreadFactor: 12,
				Bandwidth:    250,
			},
			CodeRate: "4/5",
		},
		PHYPayload: []byte{1, 2, 3, 4},
	}
	client.Send(txPacket)
}
