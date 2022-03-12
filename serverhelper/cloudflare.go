package serverhelper

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	cloudflareUpdated    = false
	cloudflareNetStrings = []string{
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"108.162.192.0/18",
		"131.0.72.0/22",
		"141.101.64.0/18",
		"162.158.0.0/15",
		"172.64.0.0/13",
		"173.245.48.0/20",
		"188.114.96.0/20",
		"190.93.240.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
	}
	cloudflareNets         []*net.IPNet
	cloudflareOnce         sync.Once
	cloudflareUpdateOnline = true
)

const (
	cloudflareAddressRangeUrl = "https://www.cloudflare.com/ips-v4"
	cloudflareRealIpHeader    = "CF-Connecting-IP"
)

func CloudflareInit() {
	cloudflareOnce.Do(func() {
		defer func() {
			if !cloudflareUpdated {
				cloudflareInitOffline()
			}
		}()

		if !cloudflareUpdateOnline {
			return
		}

		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}

		errStr := "WARN failed to get latest cloudflare ip, use predefined ones: %v"
		httpResp, err := httpClient.Get(cloudflareAddressRangeUrl)
		if err != nil {
			log.Printf(errStr, err)
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			log.Printf(errStr, fmt.Errorf("http %d", httpResp.StatusCode))
			return
		}

		respBytes, err := io.ReadAll(httpResp.Body)
		if err != nil {
			log.Printf(errStr, err)
			return
		}

		strs := strings.Split(string(respBytes), "\n")
		var result []string
		for _, s := range strs {
			s := strings.TrimSpace(s)
			result = append(result, s)
		}

		for _, str := range result {
			_, n, err := net.ParseCIDR(str)
			if err != nil {
				log.Printf("WARN cannot parse CIDR, ignored: %s %v", str, err)
				continue
			}
			cloudflareNets = append(cloudflareNets, n)
		}
		if len(cloudflareNets) == 0 {
			log.Printf(errStr, fmt.Errorf("empty result"))
			return
		}

		log.Printf("INFO get latest cloudflare ip successfully")
		cloudflareUpdated = true
	})
}

func cloudflareInitOffline() {
	log.Printf("WARN online updating is disabled")

	for _, str := range cloudflareNetStrings {
		_, n, err := net.ParseCIDR(str)
		if err != nil {
			log.Printf("WARN cannot parse CIDR, ignored: %s %v", str, err)
		} else {
			cloudflareNets = append(cloudflareNets, n)
		}
	}
}

func CloudflareRealIpFunc(r *http.Request) string {
	ipStr := DefaultRealIpFunc(r)
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "0.0.0.0"
	}

	fromCloudflare := false
	for _, n := range cloudflareNets {
		if n.Contains(ip) {
			fromCloudflare = true
			break
		}
	}

	if !fromCloudflare {
		return ipStr
	}

	realIp := r.Header.Get(cloudflareRealIpHeader)
	if realIp == "" {
		return ipStr
	}
	return realIp
}
