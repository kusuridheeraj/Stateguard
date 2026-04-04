package retention

import (
	"sort"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

type Engine struct {
	window time.Duration
}

type Snapshot struct {
	CapacityBytes int64
	UsedBytes     int64
}

type Plan struct {
	DeleteIDs []string `json:"deleteIds"`
	Reason    string   `json:"reason"`
}

func NewEngine(window time.Duration) Engine {
	return Engine{window: window}
}

func (e Engine) Evaluate(records []types.ArtifactRecord, disk Snapshot, maxDiskUsagePercent int, now time.Time) Plan {
	if len(records) == 0 {
		return Plan{Reason: "no artifacts to evaluate"}
	}

	var deleteIDs []string
	for _, record := range records {
		if e.window > 0 && now.Sub(record.CreatedAt) > e.window {
			deleteIDs = append(deleteIDs, record.ID)
		}
	}
	if len(deleteIDs) > 0 {
		return Plan{DeleteIDs: deleteIDs, Reason: "retention window exceeded"}
	}

	if disk.CapacityBytes <= 0 || maxDiskUsagePercent <= 0 {
		return Plan{Reason: "disk quota evaluation skipped"}
	}

	limit := disk.CapacityBytes * int64(maxDiskUsagePercent) / 100
	if disk.UsedBytes <= limit {
		return Plan{Reason: "disk usage within configured quota"}
	}

	sorted := make([]types.ArtifactRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})

	used := disk.UsedBytes
	for _, record := range sorted {
		if used <= limit {
			break
		}
		deleteIDs = append(deleteIDs, record.ID)
		used -= record.SizeBytes
	}

	return Plan{DeleteIDs: deleteIDs, Reason: "disk quota exceeded"}
}
