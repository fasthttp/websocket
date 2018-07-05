// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bytes"
	"net"
	"net/url"
	"time"

	"github.com/erikdubbelboer/fasthttp"
	"github.com/valyala/bytebufferpool"
)

// FastHTTPHandler receives a websocket connection after the handshake has been
// completed. This must be provided.
type FastHTTPHandler func(*Conn)

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
}

func (u *FastHTTPUpgrader) responseError(ctx *fasthttp.RequestCtx, status int, reason string) error {
	err := HandshakeError{reason}
	if u.Error != nil {
		u.Error(ctx, status, err)
	} else {
		ctx.Response.Header.Set("Sec-Websocket-Version", "13")
		ctx.Error(fasthttp.StatusMessage(status), status)
	}

	return err
}

func (u *FastHTTPUpgrader) selectSubprotocol(ctx *fasthttp.RequestCtx) []byte {
	value := bytebufferpool.Get()
	defer bytebufferpool.Put(value)

	if u.Subprotocols != nil {
		clientProtocols := parseDataHeader(ctx.Request.Header.Peek("Sec-Websocket-Protocol"))
		for _, serverProtocol := range u.Subprotocols {
			for _, clientProtocol := range clientProtocols {
				value.B = append(value.B[:0], serverProtocol...)
				if bytes.Equal(clientProtocol, value.B) {
					return clientProtocol
				}
			}
		}
	} else if ctx.Response.Header.Len() > 0 {
		return ctx.Response.Header.Peek("Sec-Websocket-Protocol")
	}

	return nil
}

// Upgrade upgrades the HTTP server connection to the WebSocket protocol.
//
// The responseHeader is included in the response to the client's upgrade
// request. Use the responseHeader to specify cookies (Set-Cookie) and the
// application negotiated subprotocol (Sec-WebSocket-Protocol).
//
// If the upgrade fails, then Upgrade replies to the client with an HTTP error
// response.
func (u *FastHTTPUpgrader) Upgrade(ctx *fasthttp.RequestCtx, handler FastHTTPHandler) error {
	const badHandshake = "websocket: the client is not using the websocket protocol: "
	value := bytebufferpool.Get()
	defer bytebufferpool.Put(value)

	if !ctx.IsGet() {
		return u.responseError(ctx, fasthttp.StatusMethodNotAllowed, badHandshake+"request method is not GET")
	}

	value.B = append(value.B[:0], "Upgrade"...)
	if !bytes.Contains(ctx.Request.Header.Peek("Connection"), value.B) {
		return u.responseError(ctx, fasthttp.StatusBadRequest, badHandshake+"'upgrade' token not found in 'Connection' header")
	}

	value.B = append(value.B[:0], "websocket"...)
	if !bytes.Contains(bytes.ToLower(ctx.Request.Header.Peek("Upgrade")), value.B) {
		return u.responseError(ctx, fasthttp.StatusBadRequest, badHandshake+"'websocket' token not found in 'Upgrade' header")
	}

	value.B = append(value.B[:0], "13"...)
	if !bytes.Contains(ctx.Request.Header.Peek("Sec-Websocket-Version"), value.B) {
		return u.responseError(ctx, fasthttp.StatusBadRequest, "websocket: unsupported version: 13 not found in 'Sec-Websocket-Version' header")
	}

	if len(ctx.Response.Header.Peek("Sec-Websocket-Extensions")) > 0 {
		return u.responseError(ctx, fasthttp.StatusInternalServerError, "websocket: application specific 'Sec-WebSocket-Extensions' headers are unsupported")
	}

	checkOrigin := u.CheckOrigin
	if checkOrigin == nil {
		checkOrigin = fastHTTPcheckSameOrigin
	}
	if !checkOrigin(ctx) {
		return u.responseError(ctx, fasthttp.StatusForbidden, "websocket: request origin not allowed by FastHTTPUpgrader.CheckOrigin")
	}

	challengeKey := ctx.Request.Header.Peek("Sec-Websocket-Key")
	if len(challengeKey) == 0 {
		return u.responseError(ctx, fasthttp.StatusBadRequest, "websocket: not a websocket handshake: `Sec-WebSocket-Key' header is missing or blank")
	}

	subprotocol := u.selectSubprotocol(ctx)
	extensions := parseDataHeader(ctx.Request.Header.Peek("Sec-WebSocket-Extensions"))

	// Negotiate PMCE
	var compress bool
	value.B = append(value.B[:0], "permessage-deflate"...)
	if u.EnableCompression {
		for _, ext := range extensions {
			if !bytes.Equal(ext, value.B) {
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
	if subprotocol != nil {
		ctx.Response.Header.SetBytesV("Sec-WebSocket-Protocol", subprotocol)
	}

	ctx.Hijack(func(netConn net.Conn) {
		c := newConn(netConn, true, u.ReadBufferSize, u.WriteBufferSize)
		if subprotocol != nil {
			c.subprotocol = string(subprotocol)
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

		handler(c)
	})

	return nil
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

// FastHTTPIsWebSocketUpgrade returns true if the client requested upgrade to the
// WebSocket protocol.
func FastHTTPIsWebSocketUpgrade(ctx *fasthttp.RequestCtx) bool {
	return bytes.Equal(ctx.Request.Header.Peek("Connection"), []byte("Upgrade")) &&
		bytes.Equal(ctx.Request.Header.Peek("Upgrade"), []byte("websocket"))
}
