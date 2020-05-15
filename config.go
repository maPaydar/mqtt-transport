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
	"net"
	"strings"
)

// Constants used throughout the service.
const (
	ChannelSeparator = '/'   // The separator character.
	maxMessageSize   = 65536 // Default Maximum message size allowed from/to the peer.
)

// Type alias for a raw config
type secretStoreConfig = map[string]interface{}

// toUsername converts an ip address to a username for Vault.
func toUsername(a net.IPAddr) string {
	return strings.Replace(
		strings.Replace(a.IP.String(), ".", "-", -1),
		":", "-", -1)
}

// New reads or creates a configuration.
func New(filename string) *Config {
	// TODO: Use viper
	return nil
}

// Config represents main configuration.
type Config struct {
	Limit      LimitConfig `json:"limit,omitempty"` // Configuration for various limits such as message size.
	ListenAddr *net.TCPAddr                         // The listen address, parsed.
}

// MaxMessageBytes returns the configured max message size, must be smaller than 64K.
func (c *Config) MaxMessageBytes() int64 {
	if c.Limit.MessageSize <= 0 || c.Limit.MessageSize > maxMessageSize {
		return maxMessageSize
	}
	return int64(c.Limit.MessageSize)
}

// Addr returns the listen address configured.
func (c *Config) Addr() *net.TCPAddr {
	return c.ListenAddr
}

// LimitConfig represents various limit configurations - such as message size.
type LimitConfig struct {
	// Maximum message size allowed from/to the client. Default if not specified is 64kB.
	MessageSize int `json:"messageSize,omitempty"`

	// The maximum messages per second allowed to be processed per client connection. This
	// effectively restricts the QpS for an individual connection.
	ReadRate int `json:"readRate,omitempty"`

	// The maximum socket write rate per connection. This does not limit QpS but instead
	// can be used to scale throughput. Defaults to 60.
	FlushRate int `json:"flushRate,omitempty"`
}
