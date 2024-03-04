//go:build go1.20 || go1.21 || go1.22

package websocket

import (
	"bufio"
	"net"
	"net/http"
)

func HijackResponse(_ *http.Request, w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w).Hijack()
}
