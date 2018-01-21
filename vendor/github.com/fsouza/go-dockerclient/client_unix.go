// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

package docker

import (
	"context"
	"net"
<<<<<<< HEAD
	"net/http"
=======
>>>>>>> Update dependencies. Mainly go-dockerclient.
)

// initializeNativeClient initializes the native Unix domain socket client on
// Unix-style operating systems
<<<<<<< HEAD
func (c *Client) initializeNativeClient(trFunc func() *http.Transport) {
	if c.endpointURL.Scheme != unixProtocol {
		return
	}
	sockPath := c.endpointURL.Path

	tr := trFunc()

	tr.Dial = func(network, addr string) (net.Conn, error) {
		return c.Dialer.Dial(unixProtocol, sockPath)
	}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return c.Dialer.Dial(unixProtocol, sockPath)
=======
func (c *Client) initializeNativeClient() {
	if c.endpointURL.Scheme != unixProtocol {
		return
	}
	socketPath := c.endpointURL.Path
	tr := defaultTransport()
	tr.Dial = func(network, addr string) (net.Conn, error) {
		return c.Dialer.Dial(unixProtocol, socketPath)
	}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return c.Dialer.Dial(unixProtocol, socketPath)
>>>>>>> Update dependencies. Mainly go-dockerclient.
	}
	c.HTTPClient.Transport = tr
}
