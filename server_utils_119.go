//go:build !go1.20 && !go1.21 && !go1.22

package websocket

import (
	"bufio"
	"net"
	"net/http"
)

func HijackResponse(r *http.Request, w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, ErrResponseHijackUnsupported
	}

	var brw *bufio.ReadWriter
	netConn, brw, err := h.Hijack()
	if err != nil {
		return nil, nil, err
	}

	return netConn, brw, nil
}
