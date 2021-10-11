package sbz

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"time"

	"github.com/cenkalti/backoff"
	H "github.com/subiz/header"
)

var (
	ErrUrlIsEmpty = errors.New("url is empty")
	ErrRetryable  = errors.New("retryable")
	ErrNot200     = errors.New("not 200")
)

// g_fhc used to send http request
var httpclient = &http.Client{Timeout: 120 * time.Second}

// Request sends http request to url, it retries automatically on
// 429 (rate limit) or 5xx error
// By default, this method will block no longer than 5 minutes, user can change
// the timeout in config paramater. The method forced to return error when
// timeout.
// If success, this method returns raw response body, an ErrNot200 is returned
// if the server don't return 2xx code.
func RequestHttp(method, path string, data interface{}, query map[string]string, timeout time.Duration) ([]byte, *H.Error) {
	url := cf.ApiURL + "/accounts/" + cf.AccountId + path

	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	// create backoff utility to do retry
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 40 * time.Second
	bo.MaxElapsedTime = timeout
	bo.Reset()

	q := neturl.Values{}
	for k, v := range query {
		q.Add(k, v)
	}
	q.Add("x-access-token", cf.ApiKey)

	rawquery := q.Encode()

	var outerr error
	var statuscode int
	var out []byte

	var body = []byte{}
	if data != nil {
		body, _ = json.Marshal(data)
	}

	err := backoff.Retry(func() error {
		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			// may be the url is invalid, exit right away
			outerr = err
			return nil
		}

		req.URL.RawQuery = rawquery

		/*
			for k, v := range header {
				req.Header.Set(k, v)
			}
		*/
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("User-Agent", "Subiz-Gun/4.016")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		res, err := httpclient.Do(req)
		if err != nil {
			// something wrong with the parameters, return nil since retry won't help
			outerr = err
			return nil
		}
		defer res.Body.Close()

		statuscode = res.StatusCode
		// trust subiz for not returning too big
		out, err = ioutil.ReadAll(res.Body)
		if err != nil {
			// return nil since retry won't help
			outerr = err
			return nil
		}

		// we don't retry on other status code (400, 300)
		if res.StatusCode != 429 && !Is5xx(res.StatusCode) {
			return nil
		}

		// retry on 429 or 5xx
		return ErrRetryable
	}, bo)
	if err != nil {
		// failed and cannot retry
		return out, H.E500(err, H.E_subiz_call_failed)
	}

	if outerr != nil {
		return out, H.E500(outerr, H.E_subiz_call_failed)
	}

	if !Is2xx(statuscode) {
		return out, H.E500(ErrNot200, H.E_subiz_call_failed)
	}

	return out, nil
}

// Is2xx return whether code is in range of (200; 299)
func Is2xx(code int) bool { return 199 < code && code < 300 }

// Is4xx tells whether code is in range of (300; 399)
func Is4xx(code int) bool { return 399 < code && code < 500 }

// Is5xx tells whether code is in range of (400; 499)
func Is5xx(code int) bool { return 499 < code && code < 600 }

/*
   if err != nil {
		// try to cast to error
		e := &H.Error{}
		if jserr := json.Unmarshal(out, e); jserr == nil {
			if e.Code != "" && e.Class != 0 {
				return out, e
			}
		}
		return out, H.E500(err, H.E_subiz_call_failed)
	}
	return out, nil
*/
