package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/roadmap"
)

func runRoadmapAdd(ctx context.Context, title, summary, status string, tags []string) error {
	manager, rm, err := loadRoadmap(ctx)
	if err != nil {
		return err
	}
	if title == "" {
		return fmt.Errorf("title is required")
	}
	item := roadmap.Item{
		Title:   title,
		Summary: summary,
		Status:  status,
		Tags:    normalizeTags(tags),
	}
	if item.Status == "" {
		item.Status = "planned"
	}
	return manager.AddItem(rm, item)
}

func runRoadmapList(ctx context.Context, status string) error {
	manager, rm, err := loadRoadmap(ctx)
	if err != nil {
		return err
	}
	logger := logging.FromContext(ctx)
	items := manager.ListItems(rm)
	if status != "" {
		status = strings.ToLower(status)
	}
	for _, item := range items {
		if status != "" && strings.ToLower(item.Status) != status {
			continue
		}
		logger.Info("roadmap item", map[string]any{
			"id":      item.ID,
			"title":   item.Title,
			"status":  item.Status,
			"summary": item.Summary,
			"tags":    strings.Join(item.Tags, ","),
			"updated": item.UpdatedAt.Format("2006-01-02"),
		})
	}
	return nil
}

func runRoadmapShow(ctx context.Context, itemID string) error {
	manager, rm, err := loadRoadmap(ctx)
	if err != nil {
		return err
	}
	item := manager.GetItem(rm, itemID)
	if item == nil {
		return fmt.Errorf("item not found: %s", itemID)
	}
	logger := logging.FromContext(ctx)
	logger.Info("roadmap item", map[string]any{
		"id":      item.ID,
		"title":   item.Title,
		"status":  item.Status,
		"summary": item.Summary,
		"tags":    strings.Join(item.Tags, ","),
		"created": item.CreatedAt.Format(timeLayout),
		"updated": item.UpdatedAt.Format(timeLayout),
	})
	for _, logEntry := range item.Logs {
		logger.Info("roadmap log", map[string]any{
			"item_id": item.ID,
			"time":    logEntry.Timestamp.Format(timeLayout),
			"note":    logEntry.Note,
			"session": logEntry.SessionID,
			"source":  logEntry.Source,
		})
	}
	return nil
}

func runRoadmapStatus(ctx context.Context, itemID, status string) error {
	manager, rm, err := loadRoadmap(ctx)
	if err != nil {
		return err
	}
	if status == "" {
		return fmt.Errorf("status is required")
	}
	return manager.UpdateStatus(rm, itemID, status)
}

func runRoadmapLog(ctx context.Context, itemID, note, sessionID, source string) error {
	manager, rm, err := loadRoadmap(ctx)
	if err != nil {
		return err
	}
	if note == "" {
		return fmt.Errorf("note is required")
	}
	entry := roadmap.LogEntry{
		Note:      note,
		SessionID: sessionID,
		Source:    source,
	}
	if entry.Source == "" {
		entry.Source = "manual"
	}
	return manager.AddLog(rm, itemID, entry)
}

func loadRoadmap(ctx context.Context) (*roadmap.Manager, *roadmap.Roadmap, error) {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)
	workspace := cfg.Sandbox.WorkspaceDir
	if workspace == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("resolve workspace: %w", err)
		}
		workspace = cwd
	}
	manager, err := roadmap.NewManager(workspace, logger)
	if err != nil {
		return nil, nil, err
	}
	rm, err := manager.Load()
	if err != nil {
		return nil, nil, err
	}
	return manager, rm, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		clean := strings.TrimSpace(strings.ToLower(tag))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	return result
}

const timeLayout = "2006-01-02 15:04:05"
