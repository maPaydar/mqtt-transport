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
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/maPaydar/mqtt-transport/pkg/config"
	"github.com/maPaydar/mqtt-transport/pkg/network/listener"
	"github.com/maPaydar/mqtt-transport/pkg/network/websocket"

	"github.com/kelindar/tcp"
)

// Service represents the main structure.
type Service struct {
	connections   int64              // The number of currently open connections.
	context       context.Context    // The context for the service.
	cancel        context.CancelFunc // The cancellation function.
	http          *http.Server       // The underlying HTTP server.
	tcp           *tcp.Server        // The underlying TCP server.
	config        *config.Config
	handler       MQTTHandler
}

// NewService creates a new service.
func NewService(listenAddr string, handler MQTTHandler) (s *Service, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	addr, _ := net.ResolveTCPAddr("tcp", listenAddr)

	s = &Service{
		context:       ctx,
		cancel:        cancel,
		http:          new(http.Server),
		tcp:           new(tcp.Server),
		handler:       handler,
		config: &config.Config{
			Limit: config.LimitConfig{
				MessageSize: 1000,
				ReadRate:    1000,
				FlushRate:   1000,
			},
			ListenAddr: addr,
		},
	}

	// Create a new HTTP request multiplexer
	mux := http.NewServeMux()

	// Attach handlers
	s.http.Handler = mux
	s.tcp.OnAccept = s.onAcceptConn

	// Attach handlers
	mux.HandleFunc("/", s.onRequest)
	return s, nil
}

// Listen starts the service.
func (s *Service) Listen() (err error) {
	defer s.Close()
	s.hookSignals()

	// Setup the listeners on both default and a secure addresses
	s.listen(s.config.ListenAddr, nil)
	// Block
	//logging.LogAction("service", "service started")
	select {}
}

// listen configures an main listener on a specified address.
func (s *Service) listen(addr *net.TCPAddr, conf *tls.Config) {

	// Create new listener
	//logging.LogTarget("service", "starting the listener", addr)
	l, err := listener.New(addr.String(), listener.Config{
		FlushRate: s.config.Limit.FlushRate,
		TLS:       conf,
	})
	if err != nil {
		panic(err)
	}

	// Set the read timeout on our mux listener
	l.SetReadTimeout(120 * time.Second)

	// Configure the matchers
	l.ServeAsync(listener.MatchHTTP(), s.http.Serve)
	l.ServeAsync(listener.MatchAny(), s.tcp.Serve)
	go l.Serve()
}

// Occurs when a new client connection is accepted.
func (s *Service) onAcceptConn(t net.Conn) {
	conn := s.newConn(t, s.config.Limit.ReadRate)
	go conn.Process()
}

// Occurs when a new HTTP request is received.
func (s *Service) onRequest(w http.ResponseWriter, r *http.Request) {
	if ws, ok := websocket.TryUpgrade(w, r); ok {
		s.onAcceptConn(ws)
		return
	}
}

// OnSignal will be called when a OS-level signal is received.
func (s *Service) onSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGTERM:
		fallthrough
	case syscall.SIGINT:
		//logging.LogAction("service", fmt.Sprintf("received signal %s, exiting...", sig.String()))
		s.Close()
		os.Exit(0)
	}
}

// OnSignal starts the signal processing and makes su
func (s *Service) hookSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range c {
			s.onSignal(sig)
		}
	}()
}

// Close closes gracefully the service.,
func (s *Service) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func dispose(resource io.Closer) {
	closable := !(resource == nil || (reflect.ValueOf(resource).Kind() == reflect.Ptr && reflect.ValueOf(resource).IsNil()))
	if closable {
		if err := resource.Close(); err != nil {
			//logging.LogError("service", "error during close", err)
		}
	}
}
