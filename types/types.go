package types

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Backend struct {
	URL       string
	IsDead    bool
	mutexLock sync.RWMutex

	ActiveConnections int64
}

type RateLimit struct {
	RequestPerSecond float64 `json:"requestPerSecond"`
	Burst            int     `json:"burst"`
}

type Config struct {
	Routes    map[string][]*Backend `json:"routes"`
	Default   string                `json:"default"`
	TimeOut   int                   `josn:"timeOut"`
	Algo      string                `json:"algo"`
	RateLimit RateLimit             `json:"ratelimit"`
}

// Backend interface lock and unlock

func (b *Backend) SetDead(dead bool) {
	b.mutexLock.Lock()
	b.IsDead = dead
	b.mutexLock.Unlock()
}

// backend interface uses to read the locks
func (b *Backend) GetDead() bool {
	b.mutexLock.RLock()
	defer b.mutexLock.RUnlock()
	return b.IsDead
}

// custome unmarshal function
func (c *Config) UnmarshalJSON(data []byte) error {
	var temp struct {
		RawRoutes map[string]interface{} `json:"routes"`
		Default   string                 `json:"default"`
		TimeOut   int                    `json:"timeOut"`
		Algo      string                 `json:"algo"`
		RateLimit RateLimit              `json:"ratelimit"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	c.Default = temp.Default
	c.TimeOut = temp.TimeOut
	c.Routes = make(map[string][]*Backend)
	c.Algo = temp.Algo
	c.RateLimit = temp.RateLimit

	for path, target := range temp.RawRoutes {
		switch v := target.(type) {
		case string:
			c.Routes[path] = []*Backend{{URL: v, IsDead: false}}
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					c.Routes[path] = append(c.Routes[path], &Backend{URL: str, IsDead: false})
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
