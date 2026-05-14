package connection

import (
	"fmt"
	"io"
	"lb4a/types"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
)

func MannualProxy(w http.ResponseWriter, r *http.Request) {
	//  Resolve where this request is going

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

	//  Connection Tracking (Only if it's not the default fallback route)
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

	// 3. Fire the request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		// This is where our Passive Health Checks will eventually trigger
		http.Error(w, "Backend server is down", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 4. Stream the backend's response back to the client
	streamResponse(w, resp)
}

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

func buildProxyRequest(originalReq *http.Request, targetURL string) (*http.Request, error) {
	proxyReq, err := http.NewRequest(originalReq.Method, targetURL, originalReq.Body)
	if err != nil {
		return nil, err
	}

	// Deep copy headers
	for key, values := range originalReq.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
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
