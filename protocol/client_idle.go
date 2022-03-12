package protocol

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"net/http"
	"sync"
	"time"
)

var (
	ErrUseAnotherIdleConn = errors.New("use another idle conn")
)

type IdleConn struct {
	d     *Dialer
	id    xid.ID
	ws    *websocket.Conn
	idle  bool
	mutex sync.RWMutex
	errCb func(*IdleConn)
}

func (d *Dialer) DialIdle(proxy, auth string, errCb func(*IdleConn)) (*IdleConn, error) {
	return d.DialIdleContext(context.Background(), proxy, auth, errCb)
}

func (d *Dialer) DialIdleContext(ctx context.Context, proxy, auth string, errCb func(*IdleConn)) (*IdleConn, error) {
	reqHeader := make(http.Header)
	reqHeader.Set(HeaderKeyAuth, auth)

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

	c := &IdleConn{
		d:     d,
		id:    id,
		ws:    ws,
		idle:  true,
		errCb: errCb,
	}
	go c.pinger()

	return c, nil
}

func (c *IdleConn) pinger() {
	n := 0
	for {
		time.Sleep(time.Second)

		c.mutex.RLock()
		idle := c.idle
		c.mutex.RUnlock()
		if !idle {
			break
		}

		n++
		if n < 27 {
			continue
		}
		n = 0

		c.mutex.Lock()
		if !idle {
			c.mutex.Unlock()
			break
		}
		err := c.ws.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			c.d.Logger.Error(fmt.Sprintf("send ping failure on idle conn: %s %v", c.id.String(), err))
			if c.errCb != nil {
				c.errCb(c)
			}
			c.idle = false
			c.mutex.Unlock()
			break
		}
		c.mutex.Unlock()
	}
}

func (c *IdleConn) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.idle = false

	return c.ws.Close()
}

type reqMessage struct {
	Protocol string `json:"protocol"`
	Target   string `json:"target"`
}

func (c *IdleConn) Id() string {
	return c.id.String()
}

func (c *IdleConn) Dial(proto, target string) (TCPConn, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.idle {
		return nil, ErrUseAnotherIdleConn
	}
	c.idle = false

	err := c.ws.WriteJSON(&reqMessage{
		Protocol: proto,
		Target:   target,
	})
	if err != nil {
		return nil, fmt.Errorf("send req failure: %v", err)
	}

	mt, buf, err := c.ws.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read resp failure: %v", err)
	}
	if mt != websocket.TextMessage {
		return nil, fmt.Errorf("unexpected resp type: %d", mt)
	}

	respStr := string(buf)
	if respStr != "ok" {
		return nil, fmt.Errorf("cannot open: %s", respStr)
	}

	cc := &connTcp{
		id:       c.id,
		ws:       c.ws,
		logger:   c.d.Logger,
		logLevel: c.d.LogLevel,
	}
	cc.init()
	go cc.pinger()

	cc.logInfof("connected %s", target)

	return cc, nil
}
