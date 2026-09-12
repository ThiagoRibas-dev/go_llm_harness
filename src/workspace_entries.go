package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type WorkspaceEntry struct {
	Path         string `json:"path"`
	Label        string `json:"label"`
	Depth        int    `json:"depth"`
	IsDir        bool   `json:"is_dir"`
	Collapsed    bool   `json:"collapsed,omitempty"`
	ModifiedNote string `json:"modified_note,omitempty"`
}

func GenerateWorkspaceEntries(rootPath string, config DirectoryScanConfig) ([]WorkspaceEntry, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return []WorkspaceEntry{}, nil
	}
	var entries []WorkspaceEntry
	if err := collectWorkspaceEntries(rootPath, "", 0, config, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []WorkspaceEntry{}
	}
	return entries, nil
}

func collectWorkspaceEntries(currentPath, relPrefix string, depth int, config DirectoryScanConfig, out *[]WorkspaceEntry) error {
	if depth > config.MaxDepth {
		return nil
	}
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}
	var dirs []os.DirEntry
	var files []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if shouldIgnore(name, config.IgnoredPatterns) {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name()) < strings.ToLower(dirs[j].Name()) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name()) })

	for _, dir := range dirs {
		name := dir.Name()
		relPath := filepath.ToSlash(filepath.Join(relPrefix, name))
		collapsed := shouldCollapse(name, config.CollapsedPatterns)
		*out = append(*out, WorkspaceEntry{
			Path:      relPath,
			Label:     name,
			Depth:     depth,
			IsDir:     true,
			Collapsed: collapsed,
		})
		if collapsed {
			continue
		}
		if err := collectWorkspaceEntries(filepath.Join(currentPath, name), relPath, depth+1, config, out); err != nil {
			return err
		}
	}

	fileLimit := config.MaxFilesPerDirectory
	for i, file := range files {
		if i >= fileLimit {
			break
		}
		name := file.Name()
		relPath := filepath.ToSlash(filepath.Join(relPrefix, name))
		note := ""
		if info, err := file.Info(); err == nil {
			modTime := info.ModTime()
			diff := time.Since(modTime)
			if diff < 1*time.Minute {
				note = "Modified < 1m ago"
			} else if diff < 60*time.Minute {
				note = "Modified " + strconv.Itoa(int(diff.Minutes())) + "m ago"
			} else if diff < 24*time.Hour {
				note = "Modified " + strconv.Itoa(int(diff.Hours())) + "h ago"
			}
		}
		*out = append(*out, WorkspaceEntry{
			Path:         relPath,
			Label:        name,
			Depth:        depth,
			IsDir:        false,
			ModifiedNote: note,
		})
	}
	return nil
}
