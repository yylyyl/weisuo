package serverhelper

import (
	"net/http"
	"testing"
)

func TestAwsCloudfrontInit(t *testing.T) {
	AwsCloudfrontInit()
	if !awsUpdated {
		t.FailNow()
	}
	t.Log(awsNetStrings)
}

func TestAwsCloudfrontRealIpFunc(t *testing.T) {
	awsUpdateOnline = false
	AwsCloudfrontInit()

	expect := "1.2.3.45"

	header := make(http.Header)
	header.Set(awsCloudfrontRealIpHeader, "1.2.3.45:777")
	ret := AwsCloudfrontRealIpFunc(&http.Request{
		RemoteAddr: "54.192.0.1:12345",
		Header:     header,
	})
	if ret != expect {
		t.Fatalf("unexpected response: %s", ret)
	}

	header = make(http.Header)
	header.Add(xffHeader, "2.2.2.2")
	header.Add(xffHeader, expect)
	ret = AwsCloudfrontRealIpFunc(&http.Request{
		RemoteAddr: "54.192.0.1:12345",
		Header:     header,
	})
	if ret != expect {
		t.Fatalf("unexpected response: %s", ret)
	}
}
