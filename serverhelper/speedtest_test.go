package serverhelper

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestSpeedTestHelper(t *testing.T) {
	go func() {
		http.HandleFunc("/speedtest", SpeedTestHelper)
		http.ListenAndServe("127.0.0.1:10080", nil)
	}()

	time.Sleep(time.Second)
	resp, err := http.DefaultClient.Get("http://127.0.0.1:10080/speedtest?size=20")
	if err != nil {
		t.Fatalf("get failure: %v", err)
	}

	r, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read failure: %v", err)
	}

	t.Logf("size: %d", len(r))
}
