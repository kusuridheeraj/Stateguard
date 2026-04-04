package artifacts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/kusuridheeraj/stateguard/pkg/types"
)

const indexFileName = "artifacts-index.json"

type Store struct {
	root      string
	indexPath string
	mu        sync.RWMutex
	records   []types.ArtifactRecord
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("artifact root path is required")
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact root: %w", err)
	}

	store := &Store{
		root:      root,
		indexPath: filepath.Join(root, indexFileName),
		records:   []types.ArtifactRecord{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Add(record types.ArtifactRecord) error {
	if record.ID == "" {
		return errors.New("artifact id is required")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)
	s.sortLocked()
	return s.saveLocked()
}

func (s *Store) List() []types.ArtifactRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]types.ArtifactRecord, len(s.records))
	copy(out, s.records)
	return out
}

func (s *Store) Summary() types.ArtifactSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var summary types.ArtifactSummary
	for _, record := range s.records {
		summary.Count++
		summary.TotalSizeBytes += record.SizeBytes
		if record.IntegrityValidated {
			summary.IntegrityReady++
		}
		if record.RestoreTested {
			summary.RestoreTested++
		}
		if record.Degraded {
			summary.DegradedArtifacts++
		}
	}

	return summary
}

func (s *Store) LatestByScope(scope string) (types.ArtifactRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.records {
		if record.Scope == scope {
			return record, true
		}
	}
	return types.ArtifactRecord{}, false
}

func (s *Store) Delete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	index := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		index[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.records[:0]
	for _, record := range s.records {
		if _, remove := index[record.ID]; remove {
			continue
		}
		filtered = append(filtered, record)
	}
	s.records = filtered
	s.sortLocked()
	return s.saveLocked()
}

func (s *Store) load() error {
	content, err := os.ReadFile(s.indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read artifact index: %w", err)
	}

	var records []types.ArtifactRecord
	if err := json.Unmarshal(content, &records); err != nil {
		return fmt.Errorf("decode artifact index: %w", err)
	}

	s.records = records
	s.sortLocked()
	return nil
}

func (s *Store) saveLocked() error {
	content, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode artifact index: %w", err)
	}
	return os.WriteFile(s.indexPath, content, 0o600)
}

func (s *Store) sortLocked() {
	sort.SliceStable(s.records, func(i, j int) bool {
		return s.records[i].CreatedAt.After(s.records[j].CreatedAt)
	})
}
