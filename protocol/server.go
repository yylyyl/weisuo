package protocol

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rs/xid"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
	"weisuo/logger"
	"weisuo/serverhelper"
)

type AuthenticatorFunc func(remoteIp, auth string) bool
type TargetFilterFunc func(remoteIp, target string) bool
type RealIpFunc func(r *http.Request) string

type Handler struct {
	WebsocketUpgrader *websocket.Upgrader
	Authenticator     AuthenticatorFunc
	TargetFilter      TargetFilterFunc
	RealIpFunc        RealIpFunc
	Logger            logger.Logger
	LogLevel          logger.LogLevel
}

func DefaultHandler() *Handler {
	return &Handler{
		WebsocketUpgrader: &websocket.Upgrader{
			HandshakeTimeout: 20 * time.Second,
			ReadBufferSize:   16 * 1024,
			WriteBufferSize:  16 * 1024,
		},
		RealIpFunc: serverhelper.DefaultRealIpFunc,
		Logger:     &logger.DefaultLogger{},
		LogLevel:   logger.LogLevelInfo,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &request{
		w:  w,
		r:  r,
		h:  h,
		id: xid.New(),
	}
	req.handle()
}

type request struct {
	w      http.ResponseWriter
	r      *http.Request
	h      *Handler
	id     xid.ID
	realIp string
}

func (req *request) handle() {
	req.realIp = req.h.RealIpFunc(req.r)
	auth := req.r.Header.Get(HeaderKeyAuth)
	proto := req.r.Header.Get(HeaderKeyProtocol)
	target := req.r.Header.Get(HeaderKeyTarget)

	req.logDebugf("headers %v", req.r.Header)
	req.logDebugf("auth [%s] proto [%s] target [%s]", auth, proto, target)

	if req.h.Authenticator != nil && !req.h.Authenticator(req.realIp, auth) {
		http.Error(req.w, "Invalid credentials", http.StatusUnauthorized)
		req.logWarnf("unauthorized %s", auth)
		return
	}

	if proto == "" && target == "" {
		req.handleIdleConn()
		return
	}

	req.handleDirectConn(proto, target)
}

func (req *request) handleIdleConn() {
	req.logDebugf("idle conn")

	respHeader := make(http.Header)
	respHeader.Set(HeaderKeyId, req.id.String())
	wsConn, err := req.h.WebsocketUpgrader.Upgrade(req.w, req.r, respHeader)
	if err != nil {
		req.logErrorf("websocket upgrade failure: %v", err)
		return
	}
	defer func() {
		err = wsConn.Close()
		req.logDebugf("close of ws underlying conn: %v", err)
	}()
	req.logDebugf("ws upgraded")

	var reqMsg reqMessage
	err = wsConn.ReadJSON(&reqMsg)
	if err != nil {
		wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "cannot read request"),
			time.Now().Add(time.Second),
		)
		req.logWarnf("cannot read request %v", err)
		return
	}

	if reqMsg.Protocol != "tcp" {
		wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "unsupported protocol"),
			time.Now().Add(time.Second),
		)
		req.logWarnf("unsupported protocol %s", reqMsg.Protocol)
		return
	}

	req.logInfof("connect %s", reqMsg.Target)
	rawConn, err := net.DialTimeout("tcp", reqMsg.Target, time.Second*10)
	if err != nil {
		wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Connection failure: %v", err)),
			time.Now().Add(time.Second),
		)
		req.logErrorf("connection failure: %v", err)
		return
	}
	req.logDebugf("connected")
	remoteConn := TCPConn(rawConn.(*net.TCPConn))
	defer remoteConn.Close()

	err = wsConn.WriteMessage(websocket.TextMessage, []byte("ok"))
	if err != nil {
		wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, fmt.Sprintf("Response failure: %v", err)),
			time.Now().Add(time.Second),
		)
		req.logErrorf("response failure: %v", err)
		return
	}

	req.handleNetwork(wsConn, remoteConn)
}

func (req *request) handleDirectConn(proto, target string) {
	if proto != "tcp" {
		http.Error(req.w, "Unsupported protocol", http.StatusBadRequest)
		req.logWarnf("unsupported protocol %s", proto)
		return
	}

	req.logInfof("connect %s", target)
	rawConn, err := net.DialTimeout("tcp", target, time.Second*10)
	if err != nil {
		http.Error(req.w, fmt.Sprintf("Connection failure: %v", err), http.StatusBadGateway)
		req.logErrorf("connection failure: %v", err)
		return
	}
	req.logDebugf("connected")
	remoteConn := TCPConn(rawConn.(*net.TCPConn))
	defer remoteConn.Close()

	respHeader := make(http.Header)
	respHeader.Set(HeaderKeyId, req.id.String())
	wsConn, err := req.h.WebsocketUpgrader.Upgrade(req.w, req.r, respHeader)
	if err != nil {
		req.logErrorf("websocket upgrade failure: %v", err)
		return
	}
	defer func() {
		err = wsConn.Close()
		req.logDebugf("close of ws underlying conn: %v", err)
	}()
	req.logDebugf("ws upgraded")

	req.handleNetwork(wsConn, remoteConn)
}

func (req *request) handleNetwork(wsConn *websocket.Conn, remoteConn TCPConn) {
	clientConn := &connTcp{
		id:       req.id,
		ws:       wsConn,
		logger:   req.h.Logger,
		logLevel: req.h.LogLevel,
	}
	clientConn.init()
	// no pinger on server

	var wg sync.WaitGroup
	var sent, received int64
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer clientConn.CloseWrite()
		var err error
		sent, err = io.Copy(clientConn, remoteConn)
		req.logDebugf("io_copy end 1: %v", err)
	}()
	go func() {
		defer wg.Done()
		defer remoteConn.CloseWrite()
		var err error
		received, err = io.Copy(remoteConn, clientConn)
		req.logDebugf("io_copy end 2: %v", err)
	}()

	wg.Wait()
	req.logInfof("connection closed, sent %d bytes, received %d bytes", sent, received)
}
