package httputils

import "net/http"

//go:generate mockgen -destination=../../mocks/mock_http_client.go -package=mocks github.com/pocwithmehul/common-go-lib/pkg/httputils HTTPClient

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
