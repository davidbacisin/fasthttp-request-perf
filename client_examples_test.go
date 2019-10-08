package fasthttp_request_perf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
)

func ExampleGetWithFastHttpManagedBuffers() {
	url := "https://golang.org/"

	// Acquire a request instance
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)

	// Acquire a response instance
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, resp.StatusCode())
		return
	}
	body := resp.Body()

	fmt.Printf("Response body is: %s", body)
}

func ExampleGetWithSelfManagedBuffers() []byte {
	url := "https://golang.org/"

	var body []byte // This buffer could be acquired from a custom buffer pool

	statusCode, body, err := fasthttp.Get(body, url)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return nil
	}
	if statusCode != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, statusCode)
		return nil
	}

	fmt.Printf("Response body is: %s", body)

	return body
}

func ExampleGetGzippedJsonWithFastHttp() {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://httpbin.org/json")
	// fasthttp does not automatically request a gzipped response. We must explicitly ask for it.
	req.Header.Set("Accept-Encoding", "gzip")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, resp.StatusCode())
		return
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		fmt.Printf("Expected content type application/json but got %s\n", contentType)
		return
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		fmt.Println("Unzipping...")
		body, _ = resp.BodyGunzip()
	} else {
		body = resp.Body()
	}

	fmt.Printf("Response body is: %s", body)
}

func ExampleGetGzippedJsonWithNetHttp() {
	req, _ := http.NewRequest(http.MethodGet, "https://httpbin.org/json", nil)
	// The built-in net/http Transport automatically requests a gzipped response
	// and also automatically unzips it for us in the body.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, resp.StatusCode)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Index(contentType, "application/json") != 0 {
		fmt.Printf("Expected content type application/json but got %s\n", contentType)
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Printf("Response body is: %s", body)
}
