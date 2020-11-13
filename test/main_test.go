package main

import (
	"github.com/ViRb3/optic-go"
	"net/http"
	"testing"
	"time"
)

func TestBypass(t *testing.T) {
	if err := doApi(opticgo.Config{
		ApiUrl:          opticgo.MustUrl("https://api.ipify.org/"),
		OpticUrl:        opticgo.MustUrl("http://localhost:8889"),
		ProxyListenAddr: "localhost:8889",
		DebugPrint:      false,
		TripFunc: func(baseTripper http.RoundTripper) http.RoundTripper {
			return customTripper{baseTripper}
		},
		InternetCheckTimeout: 10 * time.Second,
	}); err != nil {
		t.Error(err)
	}
}
