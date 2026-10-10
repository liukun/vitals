package main

import (
	"fmt"
	"strings"
	"time"

	"vitals/internal/core"
)

const (
	anomalySustain = 30 * time.Second
	alertCooldown  = 30 * time.Minute
)

type healthAlert struct {
	ID      string
	Title   string
	Message string
}

type anomalyState struct {
	since        time.Time
	lastNotified time.Time
}

type healthMonitor struct {
	cpu    anomalyState
	memory anomalyState
	disk   anomalyState
}

func (m *healthMonitor) Reset() {
	*m = healthMonitor{}
}

func (m *healthMonitor) Observe(snapshot core.Snapshot, now time.Time) []healthAlert {
	checks := []struct {
		state   *anomalyState
		active  bool
		id      string
		title   string
		message string
	}{
		{&m.memory, snapshot.MemoryPercent >= 90, "memory", "内存持续紧张", fmt.Sprintf("内存使用率已持续达到 %.0f%%", snapshot.MemoryPercent)},
		{&m.cpu, snapshot.CPUPercent >= 90, "cpu", "CPU 持续高负载", fmt.Sprintf("CPU 使用率已持续达到 %.0f%%", snapshot.CPUPercent)},
		{&m.disk, snapshot.DiskPercent >= 90, "disk", "磁盘空间不足", fmt.Sprintf("磁盘已使用 %.0f%%，建议在系统设置中管理存储空间", snapshot.DiskPercent)},
	}
	var alerts []healthAlert
	for _, check := range checks {
		if !check.active {
			check.state.since = time.Time{}
			continue
		}
		if check.state.since.IsZero() {
			check.state.since = now
			continue
		}
		if now.Sub(check.state.since) < anomalySustain ||
			(!check.state.lastNotified.IsZero() && now.Sub(check.state.lastNotified) < alertCooldown) {
			continue
		}
		check.state.lastNotified = now
		alerts = append(alerts, healthAlert{
			ID:      "health-" + check.id,
			Title:   check.title,
			Message: check.message,
		})
	}
	return alerts
}

func healthSummary(snapshot core.Snapshot, startupItems []core.StartupItem) (title, detail string, attention bool) {
	var issues []string
	if snapshot.MemoryPercent >= 90 {
		issues = append(issues, "内存紧张")
	}
	if snapshot.CPUPercent >= 90 {
		issues = append(issues, "CPU 高负载")
	}
	if snapshot.DiskPercent >= 90 {
		issues = append(issues, "磁盘空间不足")
	}
	review := 0
	for _, item := range startupItems {
		if item.NeedsReview {
			review++
		}
	}
	if review > 0 {
		issues = append(issues, fmt.Sprintf("%d 个启动项需关注", review))
	}
	if len(issues) == 0 {
		return "运行状态良好", "没有发现持续资源压力", false
	}
	return "建议检查", strings.Join(issues, " · "), true
}
