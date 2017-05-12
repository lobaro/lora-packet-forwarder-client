# LoRa Packet Forwarder Client

Enables communication with the [LoRa Packet Forarder](https://github.com/Lora-net/packet_forwarder) from Semtech.

The code is based on the [LoRa Gateway Bridge](https://github.com/brocaar/lora-gateway-bridge) which is also used in the [TTN Packet Forwarder Bridge](https://github.com/TheThingsNetwork/packet-forwarder-bridge) but targets a more generic aproach to talk to the Gateway, e.g. you can receive packets with wrong CRC and there is no dependencie to LoRaWAN specifics like `lorawan.EUI64`.


# Usage

```
func main() {
	client, err := gateway.NewClient(":1680", onNewGateway, onDeleteGateway)

	if err != nil {
		panic(err)
	}

	defer client.Close()

	go handleRxPackets(client)

	log.Info("Listening on poer :1680. Press Ctrl + c to exit.")

	// Block till program terminates
	exit = make(chan struct{})
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

```

Make sure you have a [LoRa Packet Forarder](https://github.com/Lora-net/packet_forwarder) configured
to communicate with the client endpoint. You can find example configs and code in `/examples`.

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/Lobaro/lora-packet-forwarder-client/blob/master/LICENSE).
