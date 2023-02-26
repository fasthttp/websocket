// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"html/template"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	filePeriod = 1 * time.Second
)

var (
	addr      = flag.String("addr", ":8080", "http service address")
	homeTempl = template.Must(template.New("").Parse(homeHTML))
	filename  string
	upgrader  = websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func readFileIfModified(lastMod time.Time) ([]byte, time.Time, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, lastMod, err
	}
	if !fi.ModTime().After(lastMod) {
		return nil, lastMod, nil
	}
	p, err := io.ReadFile(filename)
	if err != nil {
		return nil, fi.ModTime(), err
	}
	return p, fi.ModTime(), nil
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, lastMod time.Time) {
	lastError := ""
	pingTicker := time.NewTicker(pingPeriod)
	fileTicker := time.NewTicker(filePeriod)
	defer func() {
		pingTicker.Stop()
		fileTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-fileTicker.C:
			var p []byte
			var err error

			p, lastMod, err = readFileIfModified(lastMod)

			if err != nil {
				if s := err.Error(); s != lastError {
					lastError = s
					p = []byte(lastError)
				}
			} else {
				lastError = ""
			}

			if p != nil {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteMessage(websocket.TextMessage, p); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
		var lastMod time.Time
		if n, err := strconv.ParseInt(string(ctx.FormValue("lastMod")), 16, 64); err == nil {
			lastMod = time.Unix(0, n)
		}

		go writer(ws, lastMod)
		reader(ws)
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			log.Println(err)
		}
		return
	}
}

func serveHome(ctx *fasthttp.RequestCtx) {
	if !ctx.IsGet() {
		ctx.Error("Method not allowed", fasthttp.StatusMethodNotAllowed)
		return
	}

	ctx.SetContentType("text/html; charset=utf-8")

	p, lastMod, err := readFileIfModified(time.Time{})
	if err != nil {
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}
	var v = struct {
		Host    string
		Data    string
		LastMod string
	}{
		string(ctx.Host()),
		string(p),
		strconv.FormatInt(lastMod.UnixNano(), 16),
	}
	homeTempl.Execute(ctx, &v)
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("filename not specified")
	}
	filename = flag.Args()[0]

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/":
			serveHome(ctx)
		case "/ws":
			serveWs(ctx)
		default:
			ctx.Error("Unsupported path", fasthttp.StatusNotFound)
		}
	}
	server := fasthttp.Server{
		Name:    "EchoExample",
		Handler: requestHandler,
	}

	log.Fatal(server.ListenAndServe(*addr))
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>WebSocket Example</title>
    </head>
    <body>
        <pre id="fileData">{{.Data}}</pre>
	</body>
	<script type="text/javascript">
		(function() {
			var data = document.getElementById("fileData");
			var conn = new WebSocket("ws://{{.Host}}/ws?lastMod={{.LastMod}}");
			conn.onclose = function(evt) {
				data.textContent = 'Connection closed';
			}
			conn.onmessage = function(evt) {
				console.log('file updated');
				data.textContent = evt.data;
			}
		})();
	</script>
</html>
`
