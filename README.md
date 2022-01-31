# Fasthttp Gorilla WebSocket

[![Test status](https://github.com/fasthttp/websocket/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/fasthttp/websocket/actions?workflow=test)
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

### Protocol Compliance

The Gorilla WebSocket package passes the server tests in the [Autobahn Test
Suite](https://github.com/crossbario/autobahn-testsuite) using the application in the [examples/autobahn
subdirectory](_examples/autobahn).
