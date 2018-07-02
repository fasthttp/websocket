// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/erikdubbelboer/fasthttp"
)

// FastHTTPUpgrader specifies parameters for upgrading an HTTP connection to a
// WebSocket connection.
type FastHTTPUpgrader struct {
	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes. If a buffer
	// size is zero, then buffers allocated by the HTTP server are used. The
	// I/O buffer sizes do not limit the size of the messages that can be sent
	// or received.
	ReadBufferSize, WriteBufferSize int

	// Subprotocols specifies the server's supported protocols in order of
	// preference. If this field is set, then the Upgrade method negotiates a
	// subprotocol by selecting the first match in this list with a protocol
	// requested by the client.
	Subprotocols []string

	// Error specifies the function for generating HTTP error responses. If Error
	// is nil, then http.Error is used to generate the HTTP response.
	Error func(ctx *fasthttp.RequestCtx, status int, reason error)

	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, then a safe default is used: return false if the
	// Origin request header is present and the origin host is not equal to
	// request Host header.
	//
	// A CheckOrigin function should carefully validate the request origin to
	// prevent cross-site request forgery.
	CheckOrigin func(ctx *fasthttp.RequestCtx) bool

	// EnableCompression specify if the server should attempt to negotiate per
	// message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool

	// Handler receives a websocket connection after the handshake has been
	// completed. This must be provided.
	Handler func(*Conn)
}

func (u *FastHTTPUpgrader) returnError(ctx *fasthttp.RequestCtx, status int, reason string) (*Conn, error) {
	err := HandshakeError{reason}
	if u.Error != nil {
		u.Error(ctx, status, err)
	} else {
		ctx.Response.Header.Set("Sec-Websocket-Version", "13")
		ctx.Error(http.StatusText(status), status)
	}
	return nil, err
}

func (u *FastHTTPUpgrader) selectSubprotocol(ctx *fasthttp.RequestCtx) string {
	if u.Subprotocols != nil {
		clientProtocols := FastHTTPSubprotocols(ctx)
		for _, serverProtocol := range u.Subprotocols {
			for _, clientProtocol := range clientProtocols {
				if clientProtocol == serverProtocol {
					return clientProtocol
				}
			}
		}
	} else if ctx.Response.Header.Len() > 0 {
		return string(ctx.Response.Header.Peek("Sec-Websocket-Protocol"))
	}

	return ""
}

// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie) and the
// application negotiated subprotocol (Sec-WebSocket-Protocol).
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response.
func (u *FastHTTPUpgrader) Upgrade(ctx *fasthttp.RequestCtx) {
	const badHandshake = "websocket: the client is not using the websocket protocol: "

	if !ctx.IsGet() {
		u.returnError(ctx, fasthttp.StatusMethodNotAllowed, badHandshake+"request method is not GET")
		return
	}

	if !fastHTTPHeaderContainsValue(ctx, "Connection", "upgrade") {
		u.returnError(ctx, fasthttp.StatusBadRequest, badHandshake+"'upgrade' token not found in 'Connection' header")
		return
	}

	if !fastHTTPHeaderContainsValue(ctx, "Upgrade", "websocket") {
		u.returnError(ctx, fasthttp.StatusBadRequest, badHandshake+"'websocket' token not found in 'Upgrade' header")
		return
	}

	if !fastHTTPHeaderContainsValue(ctx, "Sec-Websocket-Version", "13") {
		u.returnError(ctx, fasthttp.StatusBadRequest, "websocket: unsupported version: 13 not found in 'Sec-Websocket-Version' header")
		return
	}

	if len(ctx.Response.Header.Peek("Sec-Websocket-Extensions")) > 0 {
		u.returnError(ctx, fasthttp.StatusInternalServerError, "websocket: application specific 'Sec-WebSocket-Extensions' headers are unsupported")
		return
	}

	checkOrigin := u.CheckOrigin
	if checkOrigin == nil {
		checkOrigin = fastHTTPcheckSameOrigin
	}
	if !checkOrigin(ctx) {
		u.returnError(ctx, fasthttp.StatusForbidden, "websocket: request origin not allowed by FastHTTPUpgrader.CheckOrigin")
		return
	}

	challengeKey := ctx.Request.Header.Peek("Sec-Websocket-Key")
	if len(challengeKey) == 0 {
		u.returnError(ctx, fasthttp.StatusBadRequest, "websocket: not a websocket handshake: `Sec-WebSocket-Key' header is missing or blank")
		return
	}

	subprotocol := u.selectSubprotocol(ctx)
	extensions := FastHTTPExtensions(ctx)

	// Negotiate PMCE
	var compress bool
	if u.EnableCompression {
		for _, ext := range extensions {
			if ext != "permessage-deflate" {
				continue
			}
			compress = true
			break
		}
	}

	ctx.SetStatusCode(fasthttp.StatusSwitchingProtocols)
	ctx.Response.Header.Set("Upgrade", "websocket")
	ctx.Response.Header.Set("Connection", "Upgrade")
	ctx.Response.Header.Set("Sec-WebSocket-Accept", computeAcceptKeyBytes(challengeKey))
	if compress {
		ctx.Response.Header.Set("Sec-WebSocket-Extensions", "permessage-deflate; server_no_context_takeover; client_no_context_takeover")
	}
	if subprotocol != "" {
		ctx.Response.Header.Set("Sec-WebSocket-Protocol", subprotocol)
	}

	ctx.Hijack(func(netConn net.Conn) {
		c := newConn(netConn, true, u.ReadBufferSize, u.WriteBufferSize)
		if subprotocol != "" {
			c.subprotocol = subprotocol
		}

		if compress {
			c.newCompressionWriter = compressNoContextTakeover
			c.newDecompressionReader = decompressNoContextTakeover
		}

		// Clear deadlines set by HTTP server.
		netConn.SetDeadline(time.Time{})

		if u.HandshakeTimeout > 0 {
			netConn.SetWriteDeadline(time.Now().Add(u.HandshakeTimeout))
		}

		u.Handler(c)
	})
}

// FastHTTPIsWebSocketUpgrade returns true if the client requested upgrade to the
// WebSocket protocol.
func FastHTTPIsWebSocketUpgrade(ctx *fasthttp.RequestCtx) bool {
	return fastHTTPHeaderContainsValue(ctx, "Connection", "upgrade") && fastHTTPHeaderContainsValue(ctx, "Upgrade", "websocket")
}

// FastHTTPSubprotocols returns the subprotocols requested by the client in the
// Sec-Websocket-Protocol header.
func FastHTTPSubprotocols(ctx *fasthttp.RequestCtx) []string {
	h := strings.TrimSpace(string(ctx.Request.Header.Peek("Sec-Websocket-Protocol")))
	if h == "" {
		return nil
	}
	protocols := strings.Split(h, ",")
	for i := range protocols {
		protocols[i] = strings.TrimSpace(protocols[i])
	}
	return protocols
}

// FastHTTPExtensions returns the subprotocols requested by the client in the
// Sec-WebSocket-Extensions header.
func FastHTTPExtensions(ctx *fasthttp.RequestCtx) []string {
	h := strings.TrimSpace(string(ctx.Request.Header.Peek("Sec-WebSocket-Extensions")))
	if h == "" {
		return nil
	}
	extensions := strings.Split(h, ";")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}
	return extensions
}

// fastHTTPcheckSameOrigin returns true if the origin is not set or is equal to the request host.
func fastHTTPcheckSameOrigin(ctx *fasthttp.RequestCtx) bool {
	origin := ctx.Request.Header.Peek("Origin")
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(string(origin))
	if err != nil {
		return false
	}
	return equalASCIIFold(u.Host, string(ctx.Host()))
}

// fastHTTPHeaderContainsValue check if match any header
func fastHTTPHeaderContainsValue(ctx *fasthttp.RequestCtx, header string, value string) bool {
	result := false
	matchKey := []byte(header)

	ctx.Request.Header.VisitAll(func(key []byte, val []byte) {
		if !result {
			if bytes.Equal(key, matchKey) {
				headerValue := string(val)
				if tokenContainsValue(headerValue, value) {
					result = true
				}
			}
		}
	})

	return result
}
