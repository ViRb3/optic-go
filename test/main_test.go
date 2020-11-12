package main

import (
	"github.com/ViRb3/optic-go"
	"testing"
	"time"
)

func TestBypass(t *testing.T) {
	if err := doApi(opticgo.Config{
		ApiUrl:               opticgo.MustUrl("https://ipleak.net/"),
		OpticUrl:             opticgo.MustUrl("http://localhost:8889"),
		ProxyListenAddr:      "localhost:8889",
		DebugPrint:           false,
		RoundTripper:         CustomTripper{},
		InternetCheckTimeout: 10 * time.Second,
	}); err != nil {
		t.Error(err)
	}
}
