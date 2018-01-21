// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package docker

import (
	"context"
	"net"
<<<<<<< HEAD
	"net/http"
=======
>>>>>>> Update dependencies. Mainly go-dockerclient.
	"time"

	"github.com/Microsoft/go-winio"
)

const namedPipeConnectTimeout = 2 * time.Second

type pipeDialer struct {
	dialFunc func(network, addr string) (net.Conn, error)
}

func (p pipeDialer) Dial(network, address string) (net.Conn, error) {
	return p.dialFunc(network, address)
}

// initializeNativeClient initializes the native Named Pipe client for Windows
<<<<<<< HEAD
func (c *Client) initializeNativeClient(trFunc func() *http.Transport) {
=======
func (c *Client) initializeNativeClient() {
>>>>>>> Update dependencies. Mainly go-dockerclient.
	if c.endpointURL.Scheme != namedPipeProtocol {
		return
	}
	namedPipePath := c.endpointURL.Path
	dialFunc := func(network, addr string) (net.Conn, error) {
		timeout := namedPipeConnectTimeout
		return winio.DialPipe(namedPipePath, &timeout)
	}
<<<<<<< HEAD
	tr := trFunc()
=======
	tr := defaultTransport()
>>>>>>> Update dependencies. Mainly go-dockerclient.
	tr.Dial = dialFunc
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialFunc(network, addr)
	}
	c.Dialer = &pipeDialer{dialFunc}
	c.HTTPClient.Transport = tr
}
