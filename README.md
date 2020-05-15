# MQTT transport layer implementation

Inspired by [emmiter-io](https://github.com/emitter-io/emitter) project.

MQTT protocol [reference](https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html).

This repo doesn't implement a message broker, just network layer of MQTT protocol.

## Run Server
```go
package main

import (
	"log"

	"github.com/maPaydar/mqtt-transport/pkg/network/mqtt"
)

type MQTTHandlerImpl struct {
}

func (m MQTTHandlerImpl) OnConnect(packet *mqtt.Connect) error {
	log.Println("OnConnect: ")
	return nil
}

func (m MQTTHandlerImpl) OnSubscribe(topic []byte) error {
	log.Println("OnSubscribe:", string(topic))
	return nil
}

func (m MQTTHandlerImpl) OnUnsubscribe(topic []byte) error {
	log.Println("OnUnsubscribe:", string(topic))
	return nil
}

func (m MQTTHandlerImpl) OnPublish(packet *mqtt.Publish) error {
	log.Println("OnPublish: ", string(packet.Topic))
	return nil
}

func main() {
	svc, err := NewService("localhost:9090", &MQTTHandlerImpl{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Service created")
	err = svc.Listen()
	if err != nil {
		log.Fatal(err)
	}
}
```

## JS mqtt client
```javascript
// client.js

var mqtt = require('mqtt');
var client = mqtt.connect('mqtt://localhost:9090');

client.on('connect', function () {
    client.subscribe('presence', function (err) {
        if (!err) {
            client.publish('presence', 'Hello mqtt')
        }
    })
});

client.on('message', function (topic, message) {
    console.log(message.toString());
    client.end();
});
```
```shell script
node client.js
```