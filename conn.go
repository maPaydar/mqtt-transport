/**********************************************************************************
* Copyright (c) 2009-2019 Misakai Ltd.
* This program is free software: you can redistribute it and/or modify it under the
* terms of the GNU Affero General Public License as published by the  Free Software
* Foundation, either version 3 of the License, or(at your option) any later version.
*
* This program is distributed  in the hope that it  will be useful, but WITHOUT ANY
* WARRANTY;  without even  the implied warranty of MERCHANTABILITY or FITNESS FOR A
* PARTICULAR PURPOSE.  See the GNU Affero General Public License  for  more details.
*
* You should have  received a copy  of the  GNU Affero General Public License along
* with this program. If not, see<http://www.gnu.org/licenses/>.
************************************************************************************/

package main

import (
	"bufio"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kelindar/rate"

	"github.com/maPaydar/mqtt-transport/pkg/network/mqtt"
)

const defaultReadRate = 100000

// Conn represents an incoming connection.
type Conn struct {
	sync.Mutex
	socket   net.Conn      // The transport used to read and write messages.
	username string        // The username provided by the client during MQTT connect.
	service  *Service      // The service for this connection.
	limit    *rate.Limiter // The read rate limiter.
	handler  MQTTHandler
}

// NewConn creates a new connection.
func (s *Service) newConn(t net.Conn, readRate int) *Conn {
	c := &Conn{
		service: s,
		socket:  t,
		handler: s.handler,
	}

	if readRate == 0 {
		readRate = defaultReadRate
	}

	c.limit = rate.New(readRate, time.Second)

	// Increment the connection counter
	atomic.AddInt64(&s.connections, 1)
	return c
}

// Process processes the messages.
func (c *Conn) Process() error {
	defer c.Close()
	reader := bufio.NewReaderSize(c.socket, 65536)
	maxSize := c.service.config.MaxMessageBytes()
	for {
		// Set read/write deadlines so we can close dangling connections
		c.socket.SetDeadline(time.Now().Add(time.Second * 120))
		if c.limit.Limit() {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		// Decode an incoming MQTT packet
		msg, err := mqtt.DecodePacket(reader, maxSize)
		if err != nil {
			return err
		}

		// Handle the receive
		if err := c.onReceive(msg); err != nil {
			return err
		}
	}
}

// onReceive handles an MQTT receive.
func (c *Conn) onReceive(msg mqtt.Message) error {
	switch msg.Type() {

	// We got an attempt to connect to MQTT.
	case mqtt.TypeOfConnect:
		var result uint8
		if c.handler.OnConnect(msg.(*mqtt.Connect)) != nil {
			result = 0x05 // Unauthorized
		}

		// Write the ack
		ack := mqtt.Connack{ReturnCode: result}
		if _, err := ack.EncodeTo(c.socket); err != nil {
			return err
		}

	// We got an attempt to subscribe to a channel.
	case mqtt.TypeOfSubscribe:
		packet := msg.(*mqtt.Subscribe)
		ack := mqtt.Suback{
			MessageID: packet.MessageID,
			Qos:       make([]uint8, 0, len(packet.Subscriptions)),
		}

		// Subscribe for each subscription
		for _, sub := range packet.Subscriptions {
			if err := c.handler.OnSubscribe(sub.Topic); err != nil {
				ack.Qos = append(ack.Qos, 0x80) // 0x80 indicate subscription failure
				//c.notifyError(err, packet.MessageID)
				continue
			}

			// Append the QoS
			ack.Qos = append(ack.Qos, sub.Qos)
		}

		// Acknowledge the subscription
		if _, err := ack.EncodeTo(c.socket); err != nil {
			return err
		}

	// We got an attempt to unsubscribe from a channel.
	case mqtt.TypeOfUnsubscribe:
		packet := msg.(*mqtt.Unsubscribe)
		ack := mqtt.Unsuback{MessageID: packet.MessageID}

		// Unsubscribe from each subscription
		for _, sub := range packet.Topics {
			if err := c.handler.OnUnsubscribe(sub.Topic); err != nil {
				//c.notifyError(err, packet.MessageID)
			}
		}

		// Acknowledge the unsubscription
		if _, err := ack.EncodeTo(c.socket); err != nil {
			return err
		}

	// We got an MQTT ping response, respond appropriately.
	case mqtt.TypeOfPingreq:
		ack := mqtt.Pingresp{}
		if _, err := ack.EncodeTo(c.socket); err != nil {
			return err
		}

	case mqtt.TypeOfDisconnect:
		return nil

	case mqtt.TypeOfPublish:
		packet := msg.(*mqtt.Publish)
		if err := c.handler.OnPublish(packet); err != nil {
			//c.notifyError(err, packet.MessageID)
		}

		// Acknowledge the publication
		if packet.Header.QOS > 0 {
			ack := mqtt.Puback{MessageID: packet.MessageID}
			if _, err := ack.EncodeTo(c.socket); err != nil {
				return err
			}
		}
	}

	return nil
}

// Close terminates the connection.
func (c *Conn) Close() error {
	if r := recover(); r != nil {
		//logging.LogAction("closing", fmt.Sprintf("panic recovered: %s \n %s", r, debug.Stack()))
	}

	// Close the transport and decrement the connection counter
	atomic.AddInt64(&c.service.connections, -1)
	//logging.LogTarget("conn", "closed", c.guid)
	return c.socket.Close()
}
