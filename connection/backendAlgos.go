package connection

import (
	"lb4a/types"
	"sync"
	"sync/atomic"
)

// routeCounters tracks the Round Robin index for each route prefix safely across goroutines
var routeCounters sync.Map

func ChooseBackend(prefix string, backends []*types.Backend, algo string) *types.Backend {
	if algo == "least-connection" || algo == "least-conn" || algo == "2" {
		return LeastConnections(backends)
	}

	return RoundRobin(prefix, backends)

}

func RoundRobin(prefix string, backends []*types.Backend) *types.Backend {
	val, _ := routeCounters.LoadOrStore(prefix, new(uint64))
	counter := val.(*uint64)

	// types.Log.Info("round-robin activated")

	// Loop through the array to find the next ALIVE server.
	for i := 0; i < len(backends); i++ {
		nextVal := atomic.AddUint64(counter, 1)
		index := (nextVal - 1) % uint64(len(backends))

		// Check if it's healthy!
		if !backends[index].GetDead() {
			// CHANGED: Return the whole struct, not just the URL string
			return backends[index]
		}
	}

	// CHANGED: If 100% of servers are dead, return nil instead of ""
	return nil
}
