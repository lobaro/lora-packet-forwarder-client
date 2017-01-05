# LoRa Packet Forwarder Client

Enables communication with the [LoRa Packet Forarder](https://github.com/Lora-net/packet_forwarder) from Semtech.

The code is based on the [LoRa Gateway Bridge](https://github.com/brocaar/lora-gateway-bridge) which is also used in the [TTN Packet Forwarder Bridge](https://github.com/TheThingsNetwork/packet-forwarder-bridge) but targets a more generic aproach to talk to the Gateway, e.g. you can receive packets with wrong CRC and there is no dependencie to LoRaWAN specifics like `lorawan.EUI64`.
