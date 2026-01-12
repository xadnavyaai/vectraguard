package roadmap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

type Roadmap struct {
	Workspace string    `json:"workspace"`
	Items     []Item    `json:"items"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Item struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Summary   string     `json:"summary"`
	Status    string     `json:"status"`
	Tags      []string   `json:"tags"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Logs      []LogEntry `json:"logs"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Note      string    `json:"note"`
	SessionID string    `json:"session_id,omitempty"`
	Source    string    `json:"source"`
}

type Manager struct {
	path      string
	workspace string
	logger    *logging.Logger
}

func NewManager(workspace string, logger *logging.Logger) (*Manager, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	baseDir, err := roadmapDir()
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s.json", hashWorkspace(abs))
	return &Manager{
		path:      filepath.Join(baseDir, name),
		workspace: abs,
		logger:    logger,
	}, nil
}

func (m *Manager) Load() (*Roadmap, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Roadmap{Workspace: m.workspace, Items: []Item{}, UpdatedAt: time.Now()}, nil
		}
		return nil, fmt.Errorf("read roadmap: %w", err)
	}

	var rm Roadmap
	if err := json.Unmarshal(data, &rm); err != nil {
		return nil, fmt.Errorf("parse roadmap: %w", err)
	}
	if rm.Workspace == "" {
		rm.Workspace = m.workspace
	}
	return &rm, nil
}

func (m *Manager) Save(roadmap *Roadmap) error {
	roadmap.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(roadmap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal roadmap: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create roadmap directory: %w", err)
	}
	if err := os.WriteFile(m.path, data, 0o644); err != nil {
		return fmt.Errorf("write roadmap: %w", err)
	}
	return nil
}

func (m *Manager) AddItem(roadmap *Roadmap, item Item) error {
	item.ID = newID()
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	roadmap.Items = append(roadmap.Items, item)
	if err := m.Save(roadmap); err != nil {
		return err
	}
	m.logger.Info("roadmap item added", map[string]any{
		"item_id": item.ID,
		"title":   item.Title,
		"status":  item.Status,
	})
	return nil
}

func (m *Manager) UpdateStatus(roadmap *Roadmap, itemID, status string) error {
	item := findItem(roadmap, itemID)
	if item == nil {
		return fmt.Errorf("item not found: %s", itemID)
	}
	item.Status = status
	item.UpdatedAt = time.Now()
	if err := m.Save(roadmap); err != nil {
		return err
	}
	m.logger.Info("roadmap status updated", map[string]any{
		"item_id": item.ID,
		"status":  status,
	})
	return nil
}

func (m *Manager) AddLog(roadmap *Roadmap, itemID string, entry LogEntry) error {
	item := findItem(roadmap, itemID)
	if item == nil {
		return fmt.Errorf("item not found: %s", itemID)
	}
	entry.Timestamp = time.Now()
	item.Logs = append(item.Logs, entry)
	item.UpdatedAt = time.Now()
	if err := m.Save(roadmap); err != nil {
		return err
	}
	m.logger.Info("roadmap log added", map[string]any{
		"item_id":  item.ID,
		"has_note": entry.Note != "",
		"session":  entry.SessionID,
	})
	return nil
}

func (m *Manager) ListItems(roadmap *Roadmap) []Item {
	items := append([]Item(nil), roadmap.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}

func (m *Manager) GetItem(roadmap *Roadmap, itemID string) *Item {
	item := findItem(roadmap, itemID)
	if item == nil {
		return nil
	}
	copy := *item
	return &copy
}

func findItem(roadmap *Roadmap, itemID string) *Item {
	for i := range roadmap.Items {
		if roadmap.Items[i].ID == itemID {
			return &roadmap.Items[i]
		}
	}
	return nil
}

func roadmapDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".vectra-guard", "roadmaps"), nil
}

func hashWorkspace(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	return fmt.Sprintf("rm-%d", time.Now().UnixNano())
}
