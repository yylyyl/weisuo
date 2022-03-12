package serverhelper

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	awsAddressRangeUrl        = "https://ip-ranges.amazonaws.com/ip-ranges.json"
	awsCloudfrontServiceName  = "CLOUDFRONT"
	awsCloudfrontRealIpHeader = "CloudFront-Viewer-Address"
	xffHeader                 = "X-Forwarded-For"
)

var (
	awsUpdated    = false
	awsNetStrings = []string{
		"120.52.22.96/27",
		"205.251.249.0/24",
		"180.163.57.128/26",
		"204.246.168.0/22",
		"18.160.0.0/15",
		"205.251.252.0/23",
		"54.192.0.0/16",
		"204.246.173.0/24",
		"54.230.200.0/21",
		"120.253.240.192/26",
		"116.129.226.128/26",
		"130.176.0.0/17",
		"108.156.0.0/14",
		"99.86.0.0/16",
		"205.251.200.0/21",
		"223.71.71.128/25",
		"13.32.0.0/15",
		"120.253.245.128/26",
		"13.224.0.0/14",
		"70.132.0.0/18",
		"15.158.0.0/16",
		"13.249.0.0/16",
		"18.238.0.0/15",
		"18.244.0.0/15",
		"205.251.208.0/20",
		"65.9.128.0/18",
		"130.176.128.0/18",
		"58.254.138.0/25",
		"54.230.208.0/20",
		"116.129.226.0/25",
		"52.222.128.0/17",
		"18.164.0.0/15",
		"64.252.128.0/18",
		"205.251.254.0/24",
		"54.230.224.0/19",
		"71.152.0.0/17",
		"216.137.32.0/19",
		"204.246.172.0/24",
		"18.172.0.0/15",
		"120.52.39.128/27",
		"118.193.97.64/26",
		"223.71.71.96/27",
		"18.154.0.0/15",
		"54.240.128.0/18",
		"205.251.250.0/23",
		"180.163.57.0/25",
		"52.46.0.0/18",
		"223.71.11.0/27",
		"52.82.128.0/19",
		"54.230.0.0/17",
		"54.230.128.0/18",
		"54.239.128.0/18",
		"130.176.224.0/20",
		"36.103.232.128/26",
		"52.84.0.0/15",
		"143.204.0.0/16",
		"144.220.0.0/16",
		"120.52.153.192/26",
		"119.147.182.0/25",
		"120.232.236.0/25",
		"54.182.0.0/16",
		"58.254.138.128/26",
		"120.253.245.192/27",
		"54.239.192.0/19",
		"18.64.0.0/14",
		"120.52.12.64/26",
		"99.84.0.0/16",
		"130.176.192.0/19",
		"52.124.128.0/17",
		"204.246.164.0/22",
		"13.35.0.0/16",
		"204.246.174.0/23",
		"36.103.232.0/25",
		"119.147.182.128/26",
		"118.193.97.128/25",
		"120.232.236.128/26",
		"204.246.176.0/20",
		"65.8.0.0/16",
		"65.9.0.0/17",
		"108.138.0.0/15",
		"120.253.241.160/27",
		"64.252.64.0/18",
		"13.113.196.64/26",
		"13.113.203.0/24",
		"52.199.127.192/26",
		"13.124.199.0/24",
		"3.35.130.128/25",
		"52.78.247.128/26",
		"13.233.177.192/26",
		"15.207.13.128/25",
		"15.207.213.128/25",
		"52.66.194.128/26",
		"13.228.69.0/24",
		"52.220.191.0/26",
		"13.210.67.128/26",
		"13.54.63.128/26",
		"99.79.169.0/24",
		"18.192.142.0/23",
		"35.158.136.0/24",
		"52.57.254.0/24",
		"13.48.32.0/24",
		"18.200.212.0/23",
		"52.212.248.0/26",
		"3.10.17.128/25",
		"3.11.53.0/24",
		"52.56.127.0/25",
		"15.188.184.0/24",
		"52.47.139.0/24",
		"18.229.220.192/26",
		"54.233.255.128/26",
		"3.231.2.0/25",
		"3.234.232.224/27",
		"3.236.169.192/26",
		"3.236.48.0/23",
		"34.195.252.0/24",
		"34.226.14.0/24",
		"13.59.250.0/26",
		"18.216.170.128/25",
		"3.128.93.0/24",
		"3.134.215.0/24",
		"52.15.127.128/26",
		"3.101.158.0/23",
		"52.52.191.128/26",
		"34.216.51.0/25",
		"34.223.12.224/27",
		"34.223.80.192/26",
		"35.162.63.192/26",
		"35.167.191.128/26",
		"44.227.178.0/24",
		"44.234.108.128/25",
		"44.234.90.252/30",
	}
	awsNets         []*net.IPNet
	awsOnce         sync.Once
	awsUpdateOnline = true
)

type awsRespRoot struct {
	Prefixes []*awsRespPrefix `json:"prefixes"`
}

type awsRespPrefix struct {
	IpPrefix string `json:"ip_prefix"`
	Service  string `json:"service"`
}

func AwsCloudfrontInit() {
	awsOnce.Do(func() {
		defer func() {
			if !awsUpdated {
				awsInitOffline()
			}
		}()

		if !awsUpdateOnline {
			return
		}

		httpClient := &http.Client{
			Timeout: time.Second * 10,
		}

		errStr := "WARN failed to get latest AWS Cloudfront ip, use predefined ones: %v"
		httpResp, err := httpClient.Get(awsAddressRangeUrl)
		if err != nil {
			log.Printf(errStr, err)
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			log.Printf(errStr, fmt.Errorf("http %d", httpResp.StatusCode))
			return
		}

		var resp awsRespRoot
		err = json.NewDecoder(httpResp.Body).Decode(&resp)
		if err != nil {
			log.Printf(errStr, err)
			return
		}
		var result []string
		for _, p := range resp.Prefixes {
			if p.Service == awsCloudfrontServiceName {
				result = append(result, p.IpPrefix)
			}
		}

		for _, str := range result {
			_, n, err := net.ParseCIDR(str)
			if err != nil {
				log.Printf("WARN cannot parse CIDR, ignored: %s %v", str, err)
				continue
			}
			awsNets = append(awsNets, n)
		}
		if len(awsNets) == 0 {
			log.Printf(errStr, fmt.Errorf("empty result"))
			return
		}

		log.Printf("INFO get latest AWS Cloudfront ip successfully")
		awsUpdated = true
	})
}

func awsInitOffline() {
	log.Printf("WARN online updating is disabled")

	for _, str := range awsNetStrings {
		_, n, err := net.ParseCIDR(str)
		if err != nil {
			log.Printf("WARN cannot parse CIDR, ignored: %s %v", str, err)
		} else {
			awsNets = append(awsNets, n)
		}
	}
}

func AwsCloudfrontRealIpFunc(r *http.Request) string {
	ipStr := DefaultRealIpFunc(r)
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "0.0.0.0"
	}

	fromAws := false
	for _, n := range awsNets {
		if n.Contains(ip) {
			fromAws = true
			break
		}
	}

	if !fromAws {
		return ipStr
	}

	realIp := r.Header.Get(awsCloudfrontRealIpHeader)
	if realIp != "" {
		colonPos := strings.Index(realIp, ":")
		if colonPos >= 7 {
			return realIp[:colonPos]
		}
	}

	xff := r.Header.Values(xffHeader)
	if len(xff) > 0 {
		return xff[len(xff)-1]
	}

	return ipStr
}
