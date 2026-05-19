package types

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Backend struct {
	URL               string `json:"URL"`
	IsDead            bool   `json:"IsDead"`
	mutexLock         sync.RWMutex
	ActiveConnections int64 `json:"ActiveConnections"`
}

type RateLimit struct {
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	Burst             int     `json:"burst"`
}

type ConnectionPool struct {
	MaxIdleConns          int    `json:"maxIdleConns"`
	MaxIdleConnsPerHost   int    `json:"maxIdleConnsPerHost"`
	IdleConnTimeout       string `json:"idleConnTimeout"`
	ResponseHeaderTimeout string `json:"responseHeaderTimeout"`
}

type TLSConfiguration struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

type Config struct {
	Routes         map[string][]*Backend `json:"routes"`
	Default        string                `json:"default"`
	TimeOut        int                   `json:"timeOut"`
	Algo           string                `json:"algo"`
	RateLimit      RateLimit             `json:"rateLimit"`
	ConnectionPool ConnectionPool        `json:"connectionPool"`
	TLS            TLSConfiguration      `json:"tls"`
	Port           string                `json:"port"`
}

func (b *Backend) SetDead(dead bool) {
	b.mutexLock.Lock()
	b.IsDead = dead
	b.mutexLock.Unlock()
}

func (b *Backend) GetDead() bool {
	b.mutexLock.RLock()
	defer b.mutexLock.RUnlock()
	return b.IsDead
}

// custome UnmarshalJSON
func (c *Config) UnmarshalJSON(data []byte) error {
	var temp struct {
		RawRoutes      map[string]interface{} `json:"routes"`
		Default        string                 `json:"default"`
		TimeOut        int                    `json:"timeOut"`
		Algo           string                 `json:"algo"`
		RateLimit      RateLimit              `json:"rateLimit"`
		ConnectionPool ConnectionPool         `json:"connectionPool"`
		TLS            TLSConfiguration       `json:"tls"`
		Port           string                 `json:"port"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	c.Default = temp.Default
	c.TimeOut = temp.TimeOut
	c.Routes = make(map[string][]*Backend)
	c.Algo = temp.Algo
	c.RateLimit = temp.RateLimit
	c.ConnectionPool = temp.ConnectionPool
	c.TLS = temp.TLS
	c.Port = temp.Port

	for path, target := range temp.RawRoutes {
		switch v := target.(type) {
		case string:
			c.Routes[path] = []*Backend{{URL: v, IsDead: false}}
		case []interface{}:
			for _, item := range v {
				// Handle array of strings
				if str, ok := item.(string); ok {
					c.Routes[path] = append(c.Routes[path], &Backend{URL: str, IsDead: false})
					continue
				}
				// Handle array of objects maps
				if obj, ok := item.(map[string]interface{}); ok {
					if urlVal, exists := obj["URL"]; exists {
						if urlStr, ok := urlVal.(string); ok {
							c.Routes[path] = append(c.Routes[path], &Backend{URL: urlStr, IsDead: false})
						}
					}
				}
			}
		default:
			return fmt.Errorf("invalid routing target format for path %s", path)
		}
	}
	return nil
}

// gloabal way to get GloabalConfigRoutes
func GetConfig() Config {
	ConfigLock.RLock()
	defer ConfigLock.RUnlock()
	return GlobalCofigRoutes
}

// gloabal way to set the config on reload or start
func SetConfig(newConfig Config) {
	ConfigLock.Lock()
	defer ConfigLock.Unlock()
	GlobalCofigRoutes = newConfig
}
