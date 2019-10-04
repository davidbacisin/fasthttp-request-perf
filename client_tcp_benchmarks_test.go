package fasthttp_request_perf

import (
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

type TcpServer struct {
	hostAddress      string
	tcpListener      net.Listener
	isRunningChannel chan struct{}
}

func handleRequest(ctx *fasthttp.RequestCtx) {
	// Fetch the query arguments from the request
	args := ctx.QueryArgs()
	if !args.Has("q") {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(args.Peek("q"))
}

func startTcpServer(b *testing.B) *TcpServer {
	hostAddress := "127.0.0.1:8542"

	// Start listening for connections
	tcpListener, err := net.Listen("tcp4", hostAddress)
	if err != nil {
		b.Fatalf("cannot listen on %q: %s", hostAddress, err)
	}

	// Use a channel to communicate if the server closes
	isRunningChannel := make(chan struct{})

	s := &TcpServer{
		hostAddress:      hostAddress,
		tcpListener:      tcpListener,
		isRunningChannel: isRunningChannel,
	}

	// Run the server as a goroutine because we need it to operate concurrently with the client
	go func() {
		if err := fasthttp.Serve(tcpListener, handleRequest); err != nil {
			b.Fatalf("error from starting server: %s", err)
		}
		close(isRunningChannel)
	}()

	return s
}

func (s *TcpServer) Stop(b *testing.B) {
	// Shutdown the server
	s.tcpListener.Close()

	// If the server doesn't stop within a second, warn us
	select {
	case <-s.isRunningChannel:
	case <-time.After(time.Second):
		b.Fatalf("server failed to stop")
	}
}

func BenchmarkNetHttpClientOverTCPToFastHttpServer(b *testing.B) {
	// Start a server
	server := startTcpServer(b)
	defer server.Stop(b)

	// Create an http.Client
	client := &http.Client{
		// Set the maximum number of idle connections equal to the current max number of processes
		Transport: &http.Transport{
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(-1),
		},
	}

	testValue := "123"
	testUrl := "http://" + server.hostAddress + "/query?q=" + testValue
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

func BenchmarkFastHttpClientOverTCPToFastHttpServer(b *testing.B) {
	// Start a server
	server := startTcpServer(b)
	defer server.Stop(b)

	// Create a fasthttp.Client
	client := &fasthttp.Client{
		// Set the maximum number of connections equal to the max number of processes
		MaxConnsPerHost: runtime.GOMAXPROCS(-1),
	}

	testValue := "123"
	testUrl := "http://" + server.hostAddress + "/query?q=" + testValue
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
