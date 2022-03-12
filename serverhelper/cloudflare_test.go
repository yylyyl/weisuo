package serverhelper

import (
	"net/http"
	"testing"
)

func TestCloudflareInit(t *testing.T) {
	CloudflareInit()
	if !cloudflareUpdated {
		t.FailNow()
	}
	t.Log(cloudflareNetStrings)
}

func TestCloudflareRealIpFunc(t *testing.T) {
	cloudflareUpdateOnline = false
	CloudflareInit()

	expect := "1.2.3.4"

	header := make(http.Header)
	header.Set(cloudflareRealIpHeader, expect)
	ret := CloudflareRealIpFunc(&http.Request{
		RemoteAddr: "103.22.200.0:12345",
		Header:     header,
	})
	if ret != expect {
		t.Fatalf("unexpected response: %s", ret)
	}
}
