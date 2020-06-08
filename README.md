# Fasthttp Gorilla WebSocket

[![Build Status](https://travis-ci.org/fasthttp/websocket.svg?branch=master)](https://travis-ci.org/fasthttp/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/fasthttp/websocket)](https://goreportcard.com/report/github.com/fasthttp/websocket)
[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/fasthttp/websocket)
[![GitHub release](https://img.shields.io/github/release/fasthttp/websocket.svg)](https://github.com/fasthttp/websocket/releases)

[Gorilla WebSocket](https://github.com/gorilla/websocket) is a [Go](http://golang.org/) implementation of the
[WebSocket](http://www.rfc-editor.org/rfc/rfc6455.txt) protocol.

This fork adds [fasthttp](https://github.com/valyala/fasthttp) support to the latest version of [gorilla/websocket](https://github.com/gorilla/websocket).

### Documentation

* [API Reference](https://pkg.go.dev/github.com/fasthttp/websocket?tab=doc)
* [Chat example](_examples/chat)
* [Command example](_examples/command)
* [Client and server example](_examples/echo)
* [File watch example](_examples/filewatch)

### Status

The Gorilla WebSocket package provides a complete and tested implementation of
the [WebSocket](http://www.rfc-editor.org/rfc/rfc6455.txt) protocol. The
package API is stable.

### Installation

```
go get github.com/fasthttp/websocket
```
But beware that this will fetch the **latest commit of the master branch** which is never purposely broken, but usually not considered stable anyway.

#### Dep
 If you're using [dep](https://github.com/golang/dep), just use `dep ensure` to add
a specific version of fasthttp/websocket including all its transitive dependencies to
your project:
 ```
dep ensure -add github.com/fasthttp/websocket@v1.4.0
```

**IMPORTANT:** [dep](https://github.com/golang/dep) is only supported until version v1.4.0. In future versions will use Go modules.

### Protocol Compliance

The Gorilla WebSocket package passes the server tests in the [Autobahn Test
Suite](https://github.com/crossbario/autobahn-testsuite) using the application in the [examples/autobahn
subdirectory](_examples/autobahn).

### Gorilla WebSocket compared with other packages

<table>
<tr>
<th></th>
<th><a href="http://godoc.org/github.com/fasthttp/websocket">github.com/fasthttp</a></th>
<th><a href="http://godoc.org/golang.org/x/net/websocket">golang.org/x/net</a></th>
</tr>
<tr>
<tr><td colspan="3"><a href="http://tools.ietf.org/html/rfc6455">RFC 6455</a> Features</td></tr>
<tr><td>Passes <a href="https://github.com/crossbario/autobahn-testsuite">Autobahn Test Suite</a></td><td><a href="https://github.com/fasthttp/websocket/tree/master/examples/autobahn">Yes</a></td><td>No</td></tr>
<tr><td>Receive <a href="https://tools.ietf.org/html/rfc6455#section-5.4">fragmented</a> message<td>Yes</td><td><a href="https://code.google.com/p/go/issues/detail?id=7632">No</a>, see note 1</td></tr>
<tr><td>Send <a href="https://tools.ietf.org/html/rfc6455#section-5.5.1">close</a> message</td><td><a href="http://godoc.org/github.com/fasthttp/websocket#hdr-Control_Messages">Yes</a></td><td><a href="https://code.google.com/p/go/issues/detail?id=4588">No</a></td></tr>
<tr><td>Send <a href="https://tools.ietf.org/html/rfc6455#section-5.5.2">pings</a> and receive <a href="https://tools.ietf.org/html/rfc6455#section-5.5.3">pongs</a></td><td><a href="http://godoc.org/github.com/fasthttp/websocket#hdr-Control_Messages">Yes</a></td><td>No</td></tr>
<tr><td>Get the <a href="https://tools.ietf.org/html/rfc6455#section-5.6">type</a> of a received data message</td><td>Yes</td><td>Yes, see note 2</td></tr>
<tr><td colspan="3">Other Features</tr></td>
<tr><td><a href="https://tools.ietf.org/html/rfc7692">Compression Extensions</a></td><td>Experimental</td><td>No</td></tr>
<tr><td>Read message using io.Reader</td><td><a href="http://godoc.org/github.com/fasthttp/websocket#Conn.NextReader">Yes</a></td><td>No, see note 3</td></tr>
<tr><td>Write message using io.WriteCloser</td><td><a href="http://godoc.org/github.com/fasthttp/websocket#Conn.NextWriter">Yes</a></td><td>No, see note 3</td></tr>
</table>

Notes:

1. Large messages are fragmented in [Chrome's new WebSocket implementation](http://www.ietf.org/mail-archive/web/hybi/current/msg10503.html).
2. The application can get the type of a received data message by implementing
   a [codec marshal](http://godoc.org/golang.org/x/net/websocket#Codec.Marshal)
   function.
3. The [go/net](https://golang.org/pkg/net/) `io.Reader` and `io.Writer` operate across WebSocket frame boundaries.
  Read returns when the input buffer is full or a frame boundary is
  encountered. Each call to Write sends a single frame message. The Gorilla
  `io.Reader` and `io.WriteCloser` operate on a single WebSocket message.
