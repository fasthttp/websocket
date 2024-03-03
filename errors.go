package websocket

import "errors"

var (
	ErrNilConn                   = errors.New("nil *Conn")
	ErrNilNetConn                = errors.New("nil net.Conn")
	ErrResponseHijackUnsupported = errors.New("websocket: response does not implement http.Hijacker")
)
