package protocol

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"weisuo/logger"
)

type TCPConn interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	CloseWrite() error
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type connTcp struct {
	id         xid.ID
	ws         *websocket.Conn
	writeMutex sync.Mutex
	readBuf    []byte
	closeWrite uint32
	closeRead  uint32
	closeOnce  sync.Once

	logger   logger.Logger
	logLevel logger.LogLevel
}

func (c *connTcp) init() {
	c.ws.SetCloseHandler(func(code int, text string) error {
		c.logDebugf("ws remote closing: %d %s", code, text)

		c.writeMutex.Lock()
		defer c.writeMutex.Unlock()

		c.setReadClosed()
		c.setWriteClosed()

		message := websocket.FormatCloseMessage(code, "")
		err := c.ws.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		c.logDebugf("ws reply close: %v", err)
		return nil
	})
}

func (c *connTcp) isReadClosed() bool {
	return atomic.LoadUint32(&c.closeRead) > 0
}
func (c *connTcp) setReadClosed() {
	atomic.StoreUint32(&c.closeRead, 1)
}
func (c *connTcp) isWriteClosed() bool {
	return atomic.LoadUint32(&c.closeWrite) > 0
}
func (c *connTcp) setWriteClosed() {
	atomic.StoreUint32(&c.closeWrite, 1)
}

func (c *connTcp) Read(buf []byte) (k int, e error) {
	//defer func() {
	//	c.logDebugf("read: %d %v", k, e)
	//}()

	if len(buf) == 0 {
		return 0, errors.New("buffer size is 0")
	}

	if c.isReadClosed() {
		return 0, errors.New("read already closed")
	}

	if len(c.readBuf) > 0 {
		n := copy(buf, c.readBuf)
		if n >= len(c.readBuf) {
			c.readBuf = nil
			c.logDebugf("read buffer emptied")
			return n, nil
		}
		c.readBuf = c.readBuf[n:]
		c.logDebugf("read buffer read")
		return n, nil
	}

	_, data, err := c.ws.ReadMessage()
	if err != nil {
		c.errHandle()
		return 0, err
	}

	if len(data) == 0 {
		c.setReadClosed()

		c.logDebugf("read EOF")
		defer c.checkClose()
		return 0, io.EOF
	}

	n := copy(buf, data)
	if len(data) > n {
		c.logDebugf("read buffer created")
		c.readBuf = data[n:]
		return n, nil
	}
	return n, nil
}

func (c *connTcp) Write(buf []byte) (k int, e error) {
	//defer func() {
	//	c.logDebugf("write: %d %v", k, e)
	//}()

	if len(buf) == 0 {
		return 0, errors.New("empty data")
	}

	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	if c.isWriteClosed() {
		return 0, errors.New("write already closed")
	}

	err := c.ws.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		c.logErrorf("write err: %v", err)
		c.errHandle()
		return 0, fmt.Errorf("write err: %v", err)
	}

	return len(buf), nil
}

func (c *connTcp) errHandle() {
	c.setWriteClosed()
	c.setReadClosed()
}

func (c *connTcp) Close() error {
	c.logDebugf("close")

	if c.isReadClosed() && c.isWriteClosed() {
		return errors.New("conn already closed")
	}

	c.setWriteClosed()
	c.setReadClosed()

	return c.close()
}

func (c *connTcp) CloseWrite() error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	if c.isWriteClosed() {
		return errors.New("write already closed")
	}
	c.setWriteClosed()

	c.logDebugf("write EOF")
	defer c.checkClose()

	err := c.ws.WriteMessage(websocket.BinaryMessage, []byte{})
	if err != nil {
		c.logErrorf("close write err: %v", err)
		c.errHandle()
		return fmt.Errorf("close write err: %v", err)
	}

	return nil
}

func (c *connTcp) checkClose() {
	if c.isReadClosed() && c.isWriteClosed() {
		go c.close()
	}
}

func (c *connTcp) close() error {
	err := errors.New("already closed")
	c.logDebugf("sending close")
	c.closeOnce.Do(func() {
		c.writeMutex.Lock()
		defer c.writeMutex.Unlock()

		_ = c.ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "ok"),
			time.Now().Add(time.Second),
		)
		err = c.ws.Close()
	})
	c.logDebugf("send close: %v", err)
	return err
}

func (c *connTcp) pinger() {
	lastPingTime := time.Now()
	for {
		time.Sleep(time.Second)

		if c.isWriteClosed() && c.isReadClosed() {
			c.logDebugf("ping stopped")
			break
		}

		if time.Now().Sub(lastPingTime) > time.Second*27 {
			err := c.ping()
			if err != nil {
				c.logDebugf("ping err, stop: %v", err)
				c.errHandle()
				break
			}
			lastPingTime = time.Now()
		}
	}
}

func (c *connTcp) ping() error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	return c.ws.WriteMessage(websocket.PingMessage, nil)
}

func (c *connTcp) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *connTcp) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *connTcp) SetDeadline(t time.Time) error {
	err := c.ws.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return c.ws.SetReadDeadline(t)
}

func (c *connTcp) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *connTcp) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
