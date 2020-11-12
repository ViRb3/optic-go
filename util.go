package opticgo

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"
)

func MustUrl(str string) *url.URL {
	parsedUrl, err := url.Parse(str)
	if err != nil {
		log.Fatalln(err)
	}
	return parsedUrl
}

// e.g. get firewall permission
func waitInternetAccess(timeout time.Duration) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	for {
		select {
		case <-ctx.Done():
			return errors.New("internet check timed out")
		default:
			if _, err := http.Get("https://google.com/"); err == nil {
				return nil
			}
			time.Sleep(1 * time.Second)
		}
	}
}
