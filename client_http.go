package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"weisuo/logger"
	"weisuo/pool"
	"weisuo/protocol"
)

type HttpProxyServer struct {
	server *http.Server
	hc     *http.Client
	pool   *pool.Pool
}

type remoteAddrMarker struct{}

var remoteAddrKey = &remoteAddrMarker{}

func runClientHttp() {
	dialer := protocol.DefaultDialer()
	dialer.LogLevel = logger.GetLevel(cfg.LogLevel)
	dialer.WsDialer.NetDialContext = getClientResolverDialer()

	s := &HttpProxyServer{}
	s.server = &http.Server{
		Addr: cfg.Listen,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.RequestURI)
			if r.Method == http.MethodConnect {
				s.handleConnect(w, r)
			} else {
				s.handleHttp(w, r)
			}
		}),
	}
	s.pool = pool.MakePool(cfg.Endpoint, cfg.Key, cfg.ClientPool, dialer)
	s.hc = s.makeHttpClient()

	err := s.server.ListenAndServe()
	log.Fatalf("client http failure: %v", err)
}

func (s *HttpProxyServer) handleConnect(w http.ResponseWriter, req *http.Request) {
	dstConn, err := s.pool.Dial("tcp", req.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("[CONNECT %s => %s] server request err: %v", req.RemoteAddr, req.Host, err)
		return
	}
	defer dstConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		log.Printf("[CONNECT %s => %s] unexpected err: cannot hijack", req.RemoteAddr, req.Host)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Internal error", http.StatusServiceUnavailable)
		log.Printf("[CONNECT %s => %s] unexpected err: failed to hijack, err: %v", req.RemoteAddr, req.Host, err)
		return
	}
	defer clientConn.Close()

	log.Printf("[CONNECT %s => %s] connected", req.RemoteAddr, req.Host)
	clientConn.Write([]byte(req.Proto + " 200 OK\r\n\r\n"))

	var wg sync.WaitGroup
	var sent, received int64
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer dstConn.CloseWrite()
		var err error
		sent, err = io.Copy(dstConn, clientConn)
		if err != nil {
			log.Printf("[CONNECT %s => %s] err 1: %v", req.RemoteAddr, req.Host, err)
		}
	}()
	go func() {
		defer wg.Done()
		defer clientConn.(*net.TCPConn).CloseWrite()
		var err error
		if err != nil {
			log.Printf("[CONNECT %s => %s] err 2: %v", req.RemoteAddr, req.Host, err)
		}
		received, err = io.Copy(clientConn, dstConn)
	}()

	wg.Wait()
	log.Printf("[CONNECT %s => %s] disconnect, sent %d received %d", req.RemoteAddr, req.Host, sent, received)
}

func (s *HttpProxyServer) makeHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				remoteAddr := ctx.Value(remoteAddrKey).(string)
				conn, err := s.pool.Dial(network, addr)
				if err != nil {
					log.Println("server request err:", remoteAddr, addr, err)
					return nil, err
				}
				log.Printf("[PROXY %s => %s] connected", remoteAddr, addr)
				return conn, nil
			},
			DisableKeepAlives: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// do not follow redirect
			return http.ErrUseLastResponse
		},
	}
}

func (s *HttpProxyServer) handleHttp(w http.ResponseWriter, req *http.Request) {
	newReq, err := http.NewRequest(req.Method, req.RequestURI, req.Body)
	if err != nil {
		log.Println("http.NewRequest err:", req, err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	for header, values := range req.Header {
		for _, value := range values {
			newReq.Header.Add(header, value)
		}
	}

	ctx := context.WithValue(context.Background(), remoteAddrKey, req.RemoteAddr)
	resp, err := s.hc.Do(newReq.WithContext(ctx))
	if err != nil {
		log.Println("httpClient.Do err:", err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}

	w.WriteHeader(resp.StatusCode)
	countRecv, _ := io.Copy(w, resp.Body)

	// TODO accurate number

	log.Printf("[PROXY %s => %s] disconnected, received %d", req.RemoteAddr, req.Host, countRecv)
}
