package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	nethttp "net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Http is an implementation of Executor that makes HTTP Requests.
// In the current implementation only JSON body requests are supported.
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

func NewHttp(logger logrus.FieldLogger) *Http {
	return &Http{
		logger:  logger,
		Timeout: 30,
	}
}

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
			return errors.New("Timeout exceeded")
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
