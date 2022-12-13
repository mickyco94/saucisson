package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	nethttp "net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Http is an implementation of Executor that makes HTTP Requests.
type Http struct {
	logger logrus.FieldLogger
	client nethttp.Client

	LogResponse bool `yaml:"log"`

	Body    string            `yaml:"body"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	URL     string            `yaml:"url"`
	Timeout int               `yaml:"timeout"`
}

// NewHttp constructs an HTTP struct with only its dependencies and defaults
// provided. Binding to configuration is done elsewhere in the struct lifecycle.
func NewHttp(logger logrus.FieldLogger, client http.Client) *Http {
	return &Http{
		logger:  logger,
		client:  client,
		Timeout: 30,
	}
}

// Execute complete a HTTP Request with parameters defined by the Http struct on which the
// execution is run.
// context.Context is used here to propagate any cancellation requests from the caller to the
// HTTP Client making the request.
// An additional timeout constraint is placed over the HTTP Request context, the length of this
// timeout is driven by the configuration defined in Http
func (http *Http) Execute(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(http.Timeout)*time.Second)
	defer cancel()

	buffer := bytes.NewBufferString(http.Body)

	request, err := nethttp.NewRequestWithContext(ctx, http.Method, http.URL, buffer)

	for k, v := range http.Headers {
		request.Header.Add(k, v)
	}

	if err != nil {
		return err
	}

	response, err := http.client.Do(request)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrTimeoutExceeded
		} else {
			return err
		}
	}

	if http.LogResponse {

		reader := json.NewDecoder(response.Body)

		var body any
		err := reader.Decode(&body)
		if err != nil {
			return err
		}

		http.logger.
			WithField("status_code", response.StatusCode).
			WithField("body", body).
			Info("HTTP Response")
	}

	return nil
}
