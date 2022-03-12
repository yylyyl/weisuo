package protocol

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"net/http"
	"time"
	"weisuo/logger"
)

type Dialer struct {
	WsDialer *websocket.Dialer
	Logger   logger.Logger
	LogLevel logger.LogLevel
}

func DefaultDialer() *Dialer {
	return &Dialer{
		WsDialer: &websocket.Dialer{
			HandshakeTimeout: 20 * time.Second,
			ReadBufferSize:   16 * 1024,
			WriteBufferSize:  16 * 1024,
		},
		Logger:   &logger.DefaultLogger{},
		LogLevel: logger.LogLevelInfo,
	}
}

func (d *Dialer) Dial(proxy, auth, proto, target string) (TCPConn, error) {
	return d.DialContext(context.Background(), proxy, auth, proto, target)
}

func (d *Dialer) DialContext(ctx context.Context, proxy, auth, proto, target string) (TCPConn, error) {
	reqHeader := make(http.Header)
	reqHeader.Set(HeaderKeyAuth, auth)
	reqHeader.Set(HeaderKeyProtocol, proto)
	reqHeader.Set(HeaderKeyTarget, target)

	ws, wsResp, err := d.WsDialer.DialContext(ctx, proxy, reqHeader)
	if err != nil {
		status := ""
		if wsResp != nil {
			status = wsResp.Status
		}
		return nil, fmt.Errorf("dial websocket failure: %v (%s)", err, status)
	}
	idString := wsResp.Header.Get(HeaderKeyId)
	id, err := xid.FromString(idString)
	if err != nil {
		return nil, fmt.Errorf("unexpected response value of %s: `%s`", HeaderKeyId, idString)
	}

	c := &connTcp{
		id:       id,
		ws:       ws,
		logger:   d.Logger,
		logLevel: d.LogLevel,
	}

	go c.pinger()

	c.logInfof("connected %s", target)

	return c, nil
}
