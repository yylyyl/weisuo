package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
)

var (
	fConfig = flag.String("config", "config.json", "path of config file")
	cfg     = Config{}
)

type Config struct {
	Listen            string `json:"listen"`
	Mode              string `json:"mode"`
	Key               string `json:"key"`
	Endpoint          string `json:"endpoint"`
	Insecure          bool   `json:"insecure"`
	TLSCert           string `json:"tls_cert"`
	TLSKey            string `json:"tls_key"`
	LogLevel          string `json:"log_level"`
	ServerPreset      string `json:"server_preset"`
	SpeedTestEndpoint string `json:"speedtest_endpoint"`
	ClientPool        uint   `json:"client_pool"`
	ClientResolver    string `json:"client_resolver"`
}

const (
	modeServer     = "server"
	modeClientNat  = "client_nat"
	modeClientHttp = "client_http"
)

func loadConfig() {
	f, err := os.Open(*fConfig)
	if err != nil {
		log.Fatalf("open config file failure: %v", err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		log.Fatalf("parse config file failure: %v", err)
	}
}

func checkConfig() {
	isClient := false
	switch cfg.Mode {
	case modeClientNat, modeClientHttp:
		isClient = true
	}

	if isClient {
		u, err := url.Parse(cfg.Endpoint)
		if err != nil {
			log.Fatalf("invalid endpont: %v", err)
		}
		switch u.Scheme {
		case "wss":
		case "ws":
			if !cfg.Insecure {
				log.Fatalf("do not use `ws` unless enable `insecure`")
			}
		default:
			log.Fatalf("invalid endpont: protocol can be either `ws` or `wss`")
		}
	}

	if cfg.Key == "" {
		log.Fatalf("empty key")
	}
}

func main() {
	flag.Parse()

	loadConfig()
	checkConfig()

	switch cfg.Mode {
	case modeServer:
		runServer()
	case modeClientHttp:
		runClientHttp()
	case modeClientNat:
		runClientNat()
	default:
		log.Fatalf("unexpected mode: %v", cfg.Mode)
	}
}
