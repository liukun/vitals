package main

import (
	"strings"
	"time"

	"vitals/internal/core"
	"vitals/internal/i18n"
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

// Observe 检测持续异常并返回需要发送的提醒，文案由 tr 提供本地化。
func (m *healthMonitor) Observe(snapshot core.Snapshot, now time.Time, tr i18n.Localizer) []healthAlert {
	checks := []struct {
		state     *anomalyState
		active    bool
		id        string
		titleKey  string
		messageFn func() string
	}{
		{
			state: &m.memory, active: snapshot.MemoryPercent >= 90, id: "memory",
			titleKey: "alert.memory_title",
			messageFn: func() string { return tr.Tf("alert.memory_body", snapshot.MemoryPercent) },
		},
		{
			state: &m.cpu, active: snapshot.CPUPercent >= 90, id: "cpu",
			titleKey: "alert.cpu_title",
			messageFn: func() string { return tr.Tf("alert.cpu_body", snapshot.CPUPercent) },
		},
		{
			state: &m.disk, active: snapshot.DiskPercent >= 90, id: "disk",
			titleKey: "alert.disk_title",
			messageFn: func() string { return tr.Tf("alert.disk_body", snapshot.DiskPercent) },
		},
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
			Title:   tr.T(check.titleKey),
			Message: check.messageFn(),
		})
	}
	return alerts
}

// healthSummary 返回状态页顶部的健康摘要文案。
func healthSummary(snapshot core.Snapshot, startupItems []core.StartupItem, tr i18n.Localizer) (title, detail string, attention bool) {
	var issues []string
	if snapshot.MemoryPercent >= 90 {
		issues = append(issues, tr.T("health.issue_memory"))
	}
	if snapshot.CPUPercent >= 90 {
		issues = append(issues, tr.T("health.issue_cpu"))
	}
	if snapshot.DiskPercent >= 90 {
		issues = append(issues, tr.T("health.issue_disk"))
	}
	review := 0
	for _, item := range startupItems {
		if item.NeedsReview {
			review++
		}
	}
	if review > 0 {
		issues = append(issues, tr.Tf("health.issue_startup", review))
	}
	if len(issues) == 0 {
		return tr.T("health.good_title"), tr.T("health.good_detail"), false
	}
	return tr.T("health.check_title"), strings.Join(issues, " · "), true
}
