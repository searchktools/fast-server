package core

import (
	"encoding/json"
	"fmt"
)

// PoolStats represents statistics for all pools
type PoolStats struct {
	Connection ConnectionPoolStats `json:"connection"`
	Context    SmartPoolStats      `json:"context"`
	Request    SmartPoolStats      `json:"request"`
	BytePool   BytePoolStats       `json:"byte_pool"`
}

type ConnectionPoolStats struct {
	Gets    uint64  `json:"gets"`
	Puts    uint64  `json:"puts"`
	HitRate float64 `json:"hit_rate"`
}

type SmartPoolStats struct {
	Gets    uint64  `json:"gets"`
	Puts    uint64  `json:"puts"`
	HitRate float64 `json:"hit_rate"`
}

type BytePoolStats struct {
	Tier512 uint64 `json:"tier_512b"`
	Tier2K  uint64 `json:"tier_2kb"`
	Tier8K  uint64 `json:"tier_8kb"`
	Tier32K uint64 `json:"tier_32kb"`
}

// GetPoolStats returns statistics for all memory pools
func (e *Engine) GetPoolStats() PoolStats {
	stats := PoolStats{}

	// Connection pool stats
	gets, puts, hitRate := e.connectionPool.Stats()
	stats.Connection = ConnectionPoolStats{
		Gets:    gets,
		Puts:    puts,
		HitRate: hitRate,
	}

	// Context pool stats
	ctxStats := e.contextPool.Stats()
	stats.Context = SmartPoolStats{
		Gets:    ctxStats.Gets,
		Puts:    ctxStats.Puts,
		HitRate: ctxStats.HitRate,
	}

	// Request pool stats
	reqStats := e.requestPool.Stats()
	stats.Request = SmartPoolStats{
		Gets:    reqStats.Gets,
		Puts:    reqStats.Puts,
		HitRate: reqStats.HitRate,
	}

	// Byte pool stats (simplified)
	stats.BytePool = BytePoolStats{
		Tier512: 0, // BytePool doesn't expose detailed stats yet
		Tier2K:  0,
		Tier8K:  0,
		Tier32K: 0,
	}

	return stats
}

// GetPoolStatsJSON returns pool statistics as JSON string
func (e *Engine) GetPoolStatsJSON() string {
	stats := e.GetPoolStats()
	data, _ := json.MarshalIndent(stats, "", "  ")
	return string(data)
}

// GetPoolStatsText returns pool statistics as human-readable text
func (e *Engine) GetPoolStatsText() string {
	stats := e.GetPoolStats()
	return fmt.Sprintf(`Memory Pool Statistics
======================

Connection Pool:
  Gets:     %d
  Puts:     %d
  Hit Rate: %.2f%%

Context Pool:
  Gets:     %d
  Puts:     %d
  Hit Rate: %.2f%%

Request Pool:
  Gets:     %d
  Puts:     %d
  Hit Rate: %.2f%%

Target: Hit Rate > 95%% for optimal performance
`,
		stats.Connection.Gets, stats.Connection.Puts, stats.Connection.HitRate*100,
		stats.Context.Gets, stats.Context.Puts, stats.Context.HitRate*100,
		stats.Request.Gets, stats.Request.Puts, stats.Request.HitRate*100,
	)
}
