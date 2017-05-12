# LoRa Packet Forwarder Client

Enables communication with the [LoRa Packet Forarder](https://github.com/Lora-net/packet_forwarder) from Semtech.

The code is based on the [LoRa Gateway Bridge](https://github.com/brocaar/lora-gateway-bridge) which is also used in the [TTN Packet Forwarder Bridge](https://github.com/TheThingsNetwork/packet-forwarder-bridge) but targets a more generic aproach to talk to the Gateway, e.g. you can receive packets with wrong CRC and there is no dependencie to LoRaWAN specifics like `lorawan.EUI64`.


# Usage

Make sure you have a [LoRa Packet Forarder](https://github.com/Lora-net/packet_forwarder) configured
to communicate with the client endpoint. You can find example configs and code in `/examples`:
[usage.go](https://github.com/Lobaro/lora-packet-forwarder-client/blob/master/examples/usage.go) and
[local_conf.json](https://github.com/Lobaro/lora-packet-forwarder-client/blob/master/examples/local_conf.json)

```
func main() {
	client, err := gateway.NewClient(
		":1680",
		func(gwMac gateway.Mac) error {
			return nil
		},
		func(gwMac gateway.Mac) error {
			return nil
		},
	)

	if err != nil {
		panic(err)
	}

	defer client.Close()

	log.Info("Waiting for gateway packet ...")
	msg := <-client.RXPacketChan()
	log.Infof("Received packet from Gateway with phy payload %v", msg.PHYPayload)

	log.Info("Exit")
}
```

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/Lobaro/lora-packet-forwarder-client/blob/master/LICENSE).
