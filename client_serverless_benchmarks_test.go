package fasthttp_request_perf

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"sync"
	"testing"

	"github.com/valyala/fasthttp"
)

type MockConn struct {
	net.Conn
	numberOfBytesRead int
	hasBeenRequested  chan struct{}
}

var mockResponseData = []byte("HTTP/1.1 200 OK\r\nContent-Type: test/plain\r\nContent-Length: 3\r\n\r\n123")
var mockServerConnectionPool = sync.Pool{
	New: func() interface{} {
		return &MockConn{
			hasBeenRequested: make(chan struct{}, 1),
		}
	},
}

func (c *MockConn) Read(b []byte) (int, error) {
	// If no bytes have been read yet, we know that the request has not been made yet
	// So, we'll wait for a request to come through
	if c.numberOfBytesRead == 0 {
		<-c.hasBeenRequested
	}

	// While there is still buffer left, copy over the response bytes
	n := 0
	for len(b) > 0 {
		if c.numberOfBytesRead == len(mockResponseData) {
			// Reset the number of bytes read for this connection
			c.numberOfBytesRead = 0
			return n, nil
		}
		// Otherwise, copy over more bytes
		n = copy(b, mockResponseData[c.numberOfBytesRead:])
		c.numberOfBytesRead += n
		b = b[n:]
	}
	return n, nil
}

func (c *MockConn) Write(b []byte) (int, error) {
	// Mark this connect as having received a request
	c.hasBeenRequested <- struct{}{}
	return len(b), nil
}

func (c *MockConn) Close() error {
	c.numberOfBytesRead = 0
	mockServerConnectionPool.Put(c)
	return nil
}

/* The Local and Remote addresses don't matter for these benchmarks because we're
 * never going to connect to an actual host. However, we want to use a static address
 * instance to avoid allocating unnecessary bytes during benchmarking.
 */
var mockServerAddr = net.TCPAddr{
	IP:   []byte{1, 2, 3, 4},
	Port: 8542,
}

func (c *MockConn) LocalAddr() net.Addr {
	return &mockServerAddr
}

func (c *MockConn) RemoteAddr() net.Addr {
	return &mockServerAddr
}

func BenchmarkNetHttpClientToMockServer(b *testing.B) {
	// Create an http.Client
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return mockServerConnectionPool.Get().(*MockConn), nil
			},
			// Set the maximum number of idle connections equal to the max number of processes
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(-1),
		},
	}

	testValue := "123"
	testUrl := "http://host.test/query"
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(testUrl)
			if err != nil {
				b.Fatalf("client get failed: %s", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("expected status code %d but got %d", http.StatusOK, resp.StatusCode)
			}
			// Read the response body
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("error while reading response body: %s", err)
			}
			if string(body) != testValue {
				b.Fatalf("expected body %q but got %q", testValue, body)
			}
		}
	})
}

func BenchmarkFastHttpClientToMockServer(b *testing.B) {
	// Create a client
	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return mockServerConnectionPool.Get().(*MockConn), nil
		},
		// Set the maximum number of idle connections equal to the max number of processes
		MaxConnsPerHost: runtime.GOMAXPROCS(-1),
	}

	testValue := "123"
	testUrl := "http://host.test/query"
	b.RunParallel(func(pb *testing.PB) {
		var buffer []byte
		for pb.Next() {
			statusCode, body, err := client.Get(buffer, testUrl)
			if err != nil {
				b.Fatalf("client get failed: %s", err)
			}
			if statusCode != fasthttp.StatusOK {
				b.Fatalf("expected status code %d but got %d", fasthttp.StatusOK, statusCode)
			}
			if string(body) != testValue {
				b.Fatalf("expected body %q but got %q", testValue, body)
			}
			buffer = body
		}
	})
}

func BenchmarkFastHttpClientWithManagedBuffersToMockServer(b *testing.B) {
	// Create a client
	client := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return mockServerConnectionPool.Get().(*MockConn), nil
		},
		// Set the maximum number of idle connections equal to the max number of processes
		MaxConnsPerHost: runtime.GOMAXPROCS(-1),
	}

	testValue := []byte("123")
	testUrl := "http://host.test/query"
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Acquire a request instance
			req := fasthttp.AcquireRequest()
			req.SetRequestURI(testUrl)

			// Acquire a response instance
			resp := fasthttp.AcquireResponse()

			err := client.Do(req, resp)
			if err != nil {
				b.Fatalf("client get failed: %s", err)
			}
			if resp.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode())
			}
			body := resp.Body()
			if !bytes.Equal(body, testValue) {
				b.Fatalf("expected body %q but got %q", testValue, body)
			}

			// Release the request and response
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}
	})
}
