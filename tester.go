package opticgo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// URL of the API that will be tested. TestDefinition.RequestUrl is relative to this URL.
	ApiUrl *url.URL
	// Proxy listen address. Default is "localhost:OPTIC_API_PORT".
	// Can specify hostname only, or hostname and port.
	// If "hostname" is given, it will be expanded into "hostname:OPTIC_API_PORT".
	// If "hostname:port" is given, it will not be altered, bypassing Optic. Useful for debugging.
	ProxyListenAddr string
	// Optic listen URL. This is the "baseUrl" in "optic.yml".
	OpticUrl *url.URL
	// Dump requests and responses to console.
	DebugPrint bool
	// Optional function to modify request before it is send to ApiUrl.
	TripFunc func(http.RoundTripper) http.RoundTripper
	// If set, Tester will wait for internet access up to this duration, and then error out.
	InternetCheckTimeout time.Duration
}

type TestDefinition struct {
	Name       string      // Test name for logging.
	Data       interface{} // Serializable JSON struct or "nil".
	RequestUrl string      // Relative to Config.ApiUrl.
	Method     string      // GET, POST, ...
}

type Tester struct {
	config       Config
	client       http.Client
	proxyStarted bool
}

func NewTester(config Config) (*Tester, error) {
	if !strings.Contains(config.ProxyListenAddr, ":") {
		opticPort := os.Getenv("OPTIC_API_PORT")
		if _, err := strconv.ParseInt(opticPort, 10, 64); err != nil {
			return nil, errors.New("bad OPTIC_API_PORT: " + err.Error())
		}
		config.ProxyListenAddr += ":" + opticPort
	}
	client := http.Client{Transport: http.DefaultTransport}
	if config.TripFunc != nil {
		client.Transport = config.TripFunc(client.Transport)
	}
	return &Tester{
		config: config,
		client: client,
	}, nil
}

// Start the proxy individually, usually for pre-setup before the tests.
func (t *Tester) StartProxy() (<-chan error, error) {
	if t.config.InternetCheckTimeout != 0 {
		if err := waitInternetAccess(t.config.InternetCheckTimeout); err != nil {
			return nil, err
		}
	}
	errChan := make(chan error)
	go func() { t.proxyListen(errChan) }()
	t.proxyStarted = true
	return errChan, nil
}

// Start the proxy and then the tests. If the proxy has already been started using StartProxy, it will skip it.
func (t *Tester) StartAll(tests []TestDefinition) (<-chan error, context.CancelFunc, error) {
	if t.config.InternetCheckTimeout != 0 {
		if err := waitInternetAccess(t.config.InternetCheckTimeout); err != nil {
			return nil, nil, err
		}
	}
	errChan := make(chan error)
	ctx, cancelFunc := context.WithCancel(context.Background())

	if !t.proxyStarted {
		go func() { t.proxyListen(errChan) }()
		t.proxyStarted = true
	}
	go func() {
		log.Printf("Defined %d tests\n", len(tests))
		for _, test := range tests {
			if ctx.Err() != nil {
				break
			}
			log.Println("Running test: " + test.RequestUrl)
			if err := t.runTest(&test); err != nil {
				errChan <- err
			}
		}
		close(errChan)
	}()
	return errChan, cancelFunc, nil
}

func (t *Tester) runTest(test *TestDefinition) error {
	testUrl, err := t.config.OpticUrl.Parse(path.Join(t.config.ApiUrl.Path, test.RequestUrl))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(test.Method, testUrl.String(), nil)
	if err != nil {
		return err
	}
	if test.Data != nil {
		req.Header.Set("Content-Type", "application/json")
		dataBytes, err := json.Marshal(test.Data)
		if err != nil {
			return err
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(dataBytes))
	}
	// send test case request to Optic
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("bad status code: %d", resp.StatusCode))
	}
	// drain the body to make sure all of it passes through Optic, otherwise we could cut it off
	if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
		return err
	}
	return nil
}

func (t *Tester) proxyListen(errChan chan<- error) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Proxy received: " + r.RequestURI)
		if err := t.proxyHandle(w, r); err != nil {
			errChan <- err
		}
		log.Println("Done with: " + r.URL.Path)
	})
	errChan <- http.ListenAndServe(t.config.ProxyListenAddr, nil)
}

func (t *Tester) proxyHandle(w http.ResponseWriter, r *http.Request) error {
	// redirect Optic request to the tested API
	r.Host = t.config.ApiUrl.Host
	r.URL = t.config.ApiUrl.ResolveReference(r.URL)
	r.RequestURI = "" // can't be set in request or client.Do will error

	if t.config.DebugPrint {
		b, err := httputil.DumpRequest(r, true)
		if err != nil {
			return err
		}
		log.Println(string(b))
	}

	resp, err := t.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if t.config.DebugPrint {
		b, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		log.Println(string(b))
	}

	// redirect API response to Optic
	for key, vals := range resp.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	return nil
}
