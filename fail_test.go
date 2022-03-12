package main

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
	"weisuo/logger"
	"weisuo/protocol"
)

func TestFailAuth(t *testing.T) {
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
	wg.Add(1)

	go func() {
		defer wg.Done()

		dialer := protocol.DefaultDialer()
		dialer.LogLevel = logger.LogLevelDebug
		_, err := dialer.Dial("ws://127.0.0.1:10080/proxy", "11111", "tcp", "a.com:1234")
		if err != nil {
			t.Logf("err %v", err)
		} else {
			errCh <- errors.New("no error occurred")
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
}
