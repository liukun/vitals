package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type appSettings struct {
	OpenAtLogin bool   `json:"-"`
	ShowCPU     bool   `json:"showCPU"`
	ShowRAM     bool   `json:"showRAM"`
	ShowGPU     bool   `json:"showGPU"`
	ShowNetwork bool   `json:"showNetwork"`
	Alerts      bool   `json:"alerts"`
	RefreshRate string `json:"refreshRate"`
}

func defaultSettings() appSettings {
	return appSettings{
		ShowCPU:     true,
		ShowRAM:     true,
		ShowGPU:     true,
		ShowNetwork: true,
		RefreshRate: "2 秒",
	}
}

func loadSettings(path string) (appSettings, error) {
	settings := defaultSettings()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("读取设置: %w", err)
	}
	settings.normalize()
	return settings, nil
}

func saveSettings(path string, settings appSettings) error {
	settings.normalize()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "settings-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (s *appSettings) normalize() {
	switch s.RefreshRate {
	case "1 秒", "2 秒", "5 秒":
	default:
		s.RefreshRate = "2 秒"
	}
	if !s.ShowCPU && !s.ShowRAM && !s.ShowGPU && !s.ShowNetwork {
		s.ShowCPU = true
		s.ShowRAM = true
		s.ShowGPU = true
		s.ShowNetwork = true
	}
}

func (s appSettings) refreshInterval() time.Duration {
	switch s.RefreshRate {
	case "1 秒":
		return time.Second
	case "5 秒":
		return 5 * time.Second
	default:
		return 2 * time.Second
	}
}

func (s appSettings) visibleMetricCount() int {
	count := 0
	for _, visible := range []bool{s.ShowCPU, s.ShowRAM, s.ShowGPU, s.ShowNetwork} {
		if visible {
			count++
		}
	}
	return count
}
