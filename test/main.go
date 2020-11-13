package main

import (
	"errors"
	"fmt"
	"github.com/ViRb3/optic-go"
	"log"
	"net/http"
	"time"
)

var defaultHeaders = map[string]string{
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/86.0.4240.183 Safari/537.36",
}

type customTripper struct {
	base http.RoundTripper
}

func (t customTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, val := range defaultHeaders {
		req.Header.Set(key, val)
	}
	return t.base.RoundTrip(req)
}

func main() {
	if err := doApi(opticgo.Config{
		ApiUrl:          opticgo.MustUrl("https://ipleak.net/"),
		OpticUrl:        opticgo.MustUrl("http://localhost:8889"),
		ProxyListenAddr: "",
		DebugPrint:      false,
		TripFunc: func(baseTripper http.RoundTripper) http.RoundTripper {
			return customTripper{baseTripper}
		},
		InternetCheckTimeout: 10 * time.Second,
	}); err != nil {
		log.Fatalln(err)
	}
}

func doApi(config opticgo.Config) error {
	tester, err := opticgo.NewTester(config)
	if err != nil {
		return err
	}

	errChan, _ := tester.Start(getTests())
	errText := ""
	for err := range errChan {
		errText += err.Error() + "\n"
	}
	if errText != "" {
		return errors.New(errText)
	}
	return nil
}

func getTests() []opticgo.TestDefinition {
	return []opticgo.TestDefinition{
		{
			"Get IP data",
			nil,
			fmt.Sprintf("/json"),
			"GET",
		},
	}
}
