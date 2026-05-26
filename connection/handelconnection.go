package connection

import (
	"crypto/tls"
	"fmt"
	"io"
	"lb4a/types"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var GlobalProxyClient *http.Client

// init function to assign a gloabl varaibale
func InitProxy() {
	GlobalProxyClient = CreateProxyClient()
}

// shareed tcp Client for Keep-alive connection
func CreateProxyClient() *http.Client {
	config := types.GetConfig()

	// Safely parse duration strings with baseline fallbacks
	idleTimeout, err := time.ParseDuration(config.ConnectionPool.IdleConnTimeout)
	if err != nil {
		idleTimeout = 90 * time.Second
	}

	headerTimeout, err := time.ParseDuration(config.ConnectionPool.ResponseHeaderTimeout)
	if err != nil {
		headerTimeout = 2 * time.Second
	}

	// Assuming TimeOut parameter in configuration maps to Milliseconds
	gatewayTimeout := time.Duration(config.TimeOut) * time.Millisecond

	var ProxyClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          config.ConnectionPool.MaxIdleConns,
			MaxIdleConnsPerHost:   config.ConnectionPool.MaxIdleConnsPerHost,
			IdleConnTimeout:       idleTimeout,
			ResponseHeaderTimeout: headerTimeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   gatewayTimeout,
		},
		Timeout: gatewayTimeout,
	}
	return ProxyClient
}

// MannualProxy To copy and forward the request and select the algorithm selected
func MannualProxy(w http.ResponseWriter, r *http.Request) {
	// Resolve where this request is going
	requestPath := r.URL.Path
	if r.URL.RawQuery != "" {
		requestPath += "?" + r.URL.RawQuery
	}

	config := types.GetConfig()

	backendPtr, targetURL := resolveTargetURL(requestPath, config.Algo)
	if backendPtr == nil && targetURL == "" {
		http.Error(w, "502 Bad Gateway - All servers offline", http.StatusBadGateway)
		return
	}

	// Connection Tracking
	if backendPtr != nil {
		atomic.AddInt64(&backendPtr.ActiveConnections, 1)
		defer atomic.AddInt64(&backendPtr.ActiveConnections, -1)
	}

	//  Clone the incoming request for the backend
	proxyReq, err := buildProxyRequest(r, targetURL)
	if err != nil {
		fmt.Println("Error creating proxy request:", err)
		http.Error(w, "Internal Gateway Error", http.StatusInternalServerError)
		return
	}

	//  Fire the request

	resp, err := GlobalProxyClient.Do(proxyReq)
	if err != nil {
		// This is where our Passive Health Checks will eventually trigger
		http.Error(w, "Backend server is down", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// INTERCEPT AND CACHE THE STREAM
	cacheKey := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery

	// Only intercept successful GET requests.
	if r.Method == http.MethodGet && resp.StatusCode == http.StatusOK {

		// Read the backend response payload stream into a byte slice exactly ONCE
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read backend body", http.StatusInternalServerError)
			return
		}

		// Inject our cache miss diagnostic tracking flag to save it on disk
		resp.Header.Set("X-Cache", "DISK_MISS_SAVED")

		// Commit it to your hybrid storage array (RAM Map + SHA256 Disk)
		ttl := 60 * time.Second
		types.GlobalHybridCache.Set(cacheKey, bodyBytes, resp.Header, resp.StatusCode, ttl)

		// Write response details to the client natively
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(bodyBytes) // Flush the bytes to the browser
		return
	}

	// 5. FALLBACK STREAMING (For POSTs, PUTs, or non-200 Errors)
	// If it shouldn't be cached, fall back to your original native stream helper
	streamResponse(w, resp)
}

// resolve the resolveTargetURL
func resolveTargetURL(requestPath string, algo string) (*types.Backend, string) {
	config := types.GetConfig()
	var keys []string
	for k := range config.Routes {
		keys = append(keys, k)
	}

	// Sort by longest prefix first to prevent route hijacking
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, prefix := range keys {
		if strings.HasPrefix(requestPath, prefix) {
			backends := config.Routes[prefix]
			selectedBackend := ChooseBackend(prefix, backends, algo)

			if selectedBackend == nil {
				return nil, ""
			}
			return selectedBackend, selectedBackend.URL + requestPath
		}
	}

	// Fallback to default
	return nil, config.Default + requestPath
}

// build the http request send to the actual backend
func buildProxyRequest(originalReq *http.Request, targetURL string) (*http.Request, error) {
	proxyReq, err := http.NewRequest(originalReq.Method, targetURL, originalReq.Body)
	if err != nil {
		return nil, err
	}

	// Deep copy headers
	for key, values := range originalReq.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
			fmt.Println(key, value)
		}
	}

	return proxyReq, nil
}

func streamResponse(w http.ResponseWriter, backendResp *http.Response) {
	// Deep copy response headers back to client
	for key, values := range backendResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Lock in the status code
	w.WriteHeader(backendResp.StatusCode)

	// Stream the body bytes
	io.Copy(w, backendResp.Body)
}
