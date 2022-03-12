package main

import (
	"log"
	"net/http"
	"weisuo/logger"
	"weisuo/protocol"
	"weisuo/serverhelper"
)

func runServer() {
	h := protocol.DefaultHandler()
	h.Authenticator = serverhelper.StaticKeyAuthenticator(cfg.Key)
	h.LogLevel = logger.GetLevel(cfg.LogLevel)
	serverPreset(h)

	mux := http.NewServeMux()
	mux.Handle(cfg.Endpoint, h)
	if cfg.SpeedTestEndpoint != "" {
		mux.HandleFunc(cfg.SpeedTestEndpoint, serverhelper.SpeedTestHelper)
	}

	var err error
	if cfg.Insecure {
		err = http.ListenAndServe(cfg.Listen, mux)
	} else {
		err = http.ListenAndServeTLS(cfg.Listen, cfg.TLSCert, cfg.TLSKey, mux)
	}

	log.Fatalf("server listen failure: %v", err)
}

func serverPreset(h *protocol.Handler) {
	switch cfg.ServerPreset {
	case "":
	case "cloudflare":
		serverhelper.CloudflareInit()
		h.RealIpFunc = serverhelper.CloudflareRealIpFunc
	case "aws_cloudfront":
		serverhelper.AwsCloudfrontInit()
		h.RealIpFunc = serverhelper.AwsCloudfrontRealIpFunc
	default:
		log.Fatalf("unexpected server preset: %s", cfg.ServerPreset)
	}
}
