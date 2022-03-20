package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"weisuo/logger"
	"weisuo/pool"
	"weisuo/protocol"
)

type NatServer struct {
	pool *pool.Pool
}

func runClientNat() {
	dialer := protocol.DefaultDialer()
	dialer.LogLevel = logger.GetLevel(cfg.LogLevel)
	dialer.WsDialer.NetDialContext = getClientResolverDialer()

	s := &NatServer{
		pool: pool.MakePool(cfg.Endpoint, cfg.Key, cfg.ClientPool, dialer),
	}

	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Fatalf("client nat listen failure: %v", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("client nat accept failure: %v", err)
		}

		go s.handleConn(conn)
	}
}

// https://gist.github.com/fangdingjun/11e5d63abe9284dc0255a574a76bbcb1
func getTcpConnOrigDst(conn net.Conn) (*net.TCPConn, net.IP, uint16, error) {
	defer conn.Close()

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, nil, 0, errors.New("not tcp conn")
	}
	tcpConnFile, err := tcpConn.File()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get tcp conn file: %v", err)
	}
	defer tcpConnFile.Close()

	newConn, err := net.FileConn(tcpConnFile)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get new tcp conn: %v", err)
	}

	const SO_ORIGINAL_DST = 80
	addr, err := syscall.GetsockoptIPv6Mreq(int(tcpConnFile.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get tcp orig dst: %v", err)
	}

	host := net.IPv4(addr.Multiaddr[4], addr.Multiaddr[5], addr.Multiaddr[6], addr.Multiaddr[7])
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	newTcpConn, ok := newConn.(*net.TCPConn)
	if !ok {
		return nil, nil, 0, errors.New("not tcp conn 2")
	}

	return newTcpConn, host, port, nil
}

func (s *NatServer) handleConn(clientConn net.Conn) {
	clientTcpConn, host, port, err := getTcpConnOrigDst(clientConn)
	if err != nil {
		log.Printf("[NAT %s => NIL] failed to get orig dst addr: %v", clientConn.RemoteAddr(), err)
		return
	}
	defer clientTcpConn.Close()

	log.Printf("[NAT %s => %s:%d] incoming", clientTcpConn.RemoteAddr(), host, port)

	dstConn, err := s.pool.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Printf("[NAT %s => %s:%d] weisuo request err: %v", clientTcpConn.RemoteAddr(), host, port, err)
		return
	}
	defer dstConn.Close()
	log.Printf("[NAT %s => %s:%d] connected", clientConn.RemoteAddr(), host, port)

	var wg sync.WaitGroup
	var sent, received int64
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer dstConn.CloseWrite()
		sent, _ = io.Copy(dstConn, clientTcpConn)
	}()
	go func() {
		defer wg.Done()
		defer clientTcpConn.CloseWrite()
		received, _ = io.Copy(clientTcpConn, dstConn)
	}()

	wg.Wait()
	log.Printf("[NAT %s => %s:%d] disconnect, sent %d received %d", clientTcpConn.RemoteAddr(), host, port, sent, received)
}
