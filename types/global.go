package types

import (
	"log/slog"
	"os"
	"sync"
)

var Routes = map[string]string{
	"/api/users1": "http://localhost:3000",
	"/api/users2": "http://localhost:3001",
}

var GlobalCofigRoutes Config
var ConfigLock sync.RWMutex

// var proxyClient = connection.CreateProxyClient()

var Log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
