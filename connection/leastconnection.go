package connection

import (
	"lb4a/types"
	"math"
	"sync/atomic"
)

func LeastConnections(backends []*types.Backend) *types.Backend {
	var backend *types.Backend

	// fmt.Println("least connection activated ")
	types.Log.Info("least connection")
	miniumConnection := int64(math.MaxInt64)

	for _, server := range backends {

		//check if its dead
		if server.GetDead() {
			continue
		}

		connections := atomic.LoadInt64(&server.ActiveConnections)

		if connections < miniumConnection {
			miniumConnection = connections
			backend = server
		}
	}

	return backend
}
