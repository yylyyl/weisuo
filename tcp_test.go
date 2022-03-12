package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
	"weisuo/logger"
	"weisuo/protocol"
)

func TestTcp(t *testing.T) {
	errCh := make(chan error)
	go func() {
		h := protocol.DefaultHandler()
		h.Authenticator = func(remoteIp, auth string) bool {
			return auth == "12345"
		}
		h.LogLevel = logger.LogLevelDebug

		mux := http.NewServeMux()
		mux.Handle("/proxy", h)
		err := http.ListenAndServe("127.0.0.1:10080", mux)
		t.Log("listen failure", err)
		errCh <- err
	}()

	time.Sleep(time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		l, err := net.Listen("tcp", "127.0.0.1:10090")
		if err != nil {
			t.Log("listen failure 2", err)
			errCh <- err
			return
		}
		tcpListener := l.(*net.TCPListener)
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			t.Log("accept failure", err)
			errCh <- err
			return
		}

		_, err = tcpConn.Write([]byte("22"))
		if err != nil {
			t.Log("write conn failure", err)
			errCh <- err
			return
		}

		buf := make([]byte, 5)
		n, err := tcpConn.Read(buf)
		if err != nil {
			t.Log("read conn failure", err)
			errCh <- err
			return
		}
		if n != 3 || string(buf[:3]) != "333" {
			t.Log("unexpected content", string(buf[:3]))
			errCh <- errors.New("unexpected content")
			return
		}

		n, err = tcpConn.Read(buf)
		if n != 0 || !errors.Is(err, io.EOF) {
			t.Log("unexpected err:", n, err)
			errCh <- err
			return
		}

		_, err = tcpConn.Write([]byte("4444"))
		if err != nil {
			t.Log("write conn failure:", err)
			errCh <- err
			return
		}

		err = tcpConn.CloseWrite()
		if err != nil {
			t.Log("close write of conn failure:", err)
			errCh <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		dialer := protocol.DefaultDialer()
		dialer.LogLevel = logger.LogLevelDebug
		conn, err := dialer.Dial("ws://127.0.0.1:10080/proxy", "12345", "tcp", "127.0.0.1:10090")
		if err != nil {
			t.Log("client connect failure", err)
			errCh <- err
			return
		}
		buf := make([]byte, 5)
		n, err := conn.Read(buf)
		if err != nil {
			t.Log("client read err:", err)
			errCh <- err
			return
		}
		if n != 2 || string(buf[:2]) != "22" {
			t.Log("client read data unexpected", string(buf[:2]))
			errCh <- errors.New("client read data")
			return
		}

		_, err = conn.Write([]byte("333"))
		if err != nil {
			t.Log("client write err:", err)
			errCh <- err
			return
		}

		err = conn.CloseWrite()
		if err != nil {
			t.Log("client close write err:", err)
			errCh <- err
			return
		}

		n, err = conn.Read(buf)
		if err != nil {
			t.Log("client read err:", err)
			errCh <- err
			return
		}
		if n != 4 || string(buf[:4]) != "4444" {
			t.Log("client read data unexpected", string(buf[:2]))
			errCh <- errors.New("client read data")
			return
		}

		n, err = conn.Read(buf)
		if n != 0 || !errors.Is(err, io.EOF) {
			t.Log("client read unexpected err", n, err)
			errCh <- err
			return
		}
	}()

	doneCh := make(chan int)
	go func() {
		wg.Wait()
		doneCh <- 1
	}()

	select {
	case err := <-errCh:
		t.Fatalf("failure: %v", err)
	case <-doneCh:
		t.Log("ok")
	}

	time.Sleep(time.Second)
}

func TestTcpIdle(t *testing.T) {
	errCh := make(chan error)
	go func() {
		h := protocol.DefaultHandler()
		h.Authenticator = func(remoteIp, auth string) bool {
			return auth == "12345"
		}
		h.LogLevel = logger.LogLevelDebug

		mux := http.NewServeMux()
		mux.Handle("/proxy", h)
		err := http.ListenAndServe("127.0.0.1:10080", mux)
		t.Log("listen failure", err)
		errCh <- err
	}()

	time.Sleep(time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		l, err := net.Listen("tcp", "127.0.0.1:10090")
		if err != nil {
			t.Log("listen failure 2", err)
			errCh <- err
			return
		}
		tcpListener := l.(*net.TCPListener)
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			t.Log("accept failure", err)
			errCh <- err
			return
		}

		_, err = tcpConn.Write([]byte("22"))
		if err != nil {
			t.Log("write conn failure", err)
			errCh <- err
			return
		}

		buf := make([]byte, 5)
		n, err := tcpConn.Read(buf)
		if err != nil {
			t.Log("read conn failure", err)
			errCh <- err
			return
		}
		if n != 3 || string(buf[:3]) != "333" {
			t.Log("unexpected content", string(buf[:3]))
			errCh <- errors.New("unexpected content")
			return
		}

		n, err = tcpConn.Read(buf)
		if n != 0 || !errors.Is(err, io.EOF) {
			t.Log("unexpected err:", n, err)
			errCh <- err
			return
		}

		_, err = tcpConn.Write([]byte("4444"))
		if err != nil {
			t.Log("write conn failure:", err)
			errCh <- err
			return
		}

		err = tcpConn.CloseWrite()
		if err != nil {
			t.Log("close write of conn failure:", err)
			errCh <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		dialer := protocol.DefaultDialer()
		dialer.LogLevel = logger.LogLevelDebug
		idleConn, err := dialer.DialIdle("ws://127.0.0.1:10080/proxy", "12345", nil)
		if err != nil {
			t.Log("client connect idle failure", err)
			errCh <- err
			return
		}

		conn, err := idleConn.Dial("tcp", "127.0.0.1:10090")
		if err != nil {
			t.Log("client connect failure:", err)
			errCh <- err
			return
		}

		buf := make([]byte, 5)
		n, err := conn.Read(buf)
		if err != nil {
			t.Log("client read err:", err)
			errCh <- err
			return
		}
		if n != 2 || string(buf[:2]) != "22" {
			t.Log("client read data unexpected", string(buf[:2]))
			errCh <- errors.New("client read data")
			return
		}

		_, err = conn.Write([]byte("333"))
		if err != nil {
			t.Log("client write err:", err)
			errCh <- err
			return
		}

		err = conn.CloseWrite()
		if err != nil {
			t.Log("client close write err:", err)
			errCh <- err
			return
		}

		n, err = conn.Read(buf)
		if err != nil {
			t.Log("client read err:", err)
			errCh <- err
			return
		}
		if n != 4 || string(buf[:4]) != "4444" {
			t.Log("client read data unexpected", string(buf[:2]))
			errCh <- errors.New("client read data")
			return
		}

		n, err = conn.Read(buf)
		if n != 0 || !errors.Is(err, io.EOF) {
			t.Log("client read unexpected err", n, err)
			errCh <- err
			return
		}
	}()

	doneCh := make(chan int)
	go func() {
		wg.Wait()
		doneCh <- 1
	}()

	select {
	case err := <-errCh:
		t.Fatalf("failure: %v", err)
	case <-doneCh:
		t.Log("ok")
	}

	time.Sleep(time.Second)
}
