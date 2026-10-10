package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"vitals/internal/core"
)

const startupSettingsURL = "x-apple.systempreferences:com.apple.LoginItems-Settings.extension"

var (
	iconPulse   = vectorIcon(`<path d="M3 12h4l2-7 4 14 2-7h6"/>`)
	iconMemory  = vectorIcon(`<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 1v4M15 1v4M9 19v4M15 19v4M1 9h4M1 15h4M19 9h4M19 15h4"/>`)
	iconGPU     = vectorIcon(`<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 9h8v6H8zM7 22h10"/>`)
	iconNetwork = vectorIcon(`<path d="M12 20h.01M5 12.6a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M2 9a15 15 0 0 1 20 0"/>`)
	iconBack    = vectorIcon(`<path d="m15 18-6-6 6-6"/>`)
	iconRocket  = vectorIcon(`<path d="M4.5 16.5c-1.5 1.3-2 5-2 5s3.7-.5 5-2M9 15l-3-3c2.2-5.2 6.1-8.7 12-9l3 3c-.3 5.9-3.8 9.8-9 12l-3-3zM15 9h.01M9 18l-3 3"/>`)
	iconRefresh = vectorIcon(`<path d="M20 12a8 8 0 1 1-2.3-5.7L20 8M20 3v5h-5"/>`)
	iconBattery = vectorIcon(`<rect x="2" y="7" width="18" height="10" rx="2"/><path d="M22 10v4M6 10v4M10 10v4M14 10v4"/>`)
	iconShield  = vectorIcon(`<path d="M12 3 5 6v5c0 4.6 2.8 8.1 7 10 4.2-1.9 7-5.4 7-10V6l-7-3z"/><path d="m9 12 2 2 4-4"/>`)
)

func vectorIcon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

func (a *application) view(c *ui.Context) {
	if c.Shortcut(0, ui.KeyEscape) || c.Shortcut(ui.Cmd, ui.KeyW) {
		if a.win != nil {
			a.win.Hide()
		}
	}
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygoQuit)
	t := c.Theme()
	c.Root().Background(ui.Transparent)
	surface := t.Background
	if c.Vibrancy() {
		surface = surface.Alpha(0.92)
	}
	ui.Column(c).Fill().Margin(1).Radius(10).Clip().Background(surface).Border(1, t.Border).Children(func() {
		a.panelHeader(c)
		if a.page == 2 {
			ui.Column(c).Grow(1).MinHeight(0).Padding(12).Children(func() {
				a.uninstaller(c)
			})
		} else {
			ui.Scroll(c).Grow(1).Padding(12).Gap(12).Children(func() {
				switch a.page {
				case 0:
					a.overview(c)
				case 1:
					a.memory(c)
				case 3:
					a.settingsView(c)
				}
			})
		}
	})
	errorResult := -1
	if a.errorSettings {
		errorResult = ui.AlertDialog(c, &a.errorOpen, "需要完全磁盘访问", a.actionError, "取消", "打开设置")
	} else {
		errorResult = ui.AlertDialog(c, &a.errorOpen, "操作未完成", a.actionError, "知道了")
	}
	if errorResult >= 0 {
		openSettings := a.errorSettings && errorResult == 1
		a.actionError = ""
		a.errorSettings = false
		if openSettings {
			if err := mygo.Shell.OpenExternal("x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"); err != nil {
				a.showActionError(err)
			}
		}
	}
	if ui.AlertDialog(
		c,
		&a.confirmMemory,
		"释放 “"+a.localizedApplicationName(a.pendingMemory.AppPath, a.pendingMemory.Name)+"” 的内存？",
		fmt.Sprintf("这会正常退出该应用及其 %d 个进程，未保存的内容可能丢失。", a.pendingMemory.ProcessCount),
		"取消",
		"退出并释放",
	) == 1 {
		a.releaseApplicationMemory()
	}
	if ui.AlertDialog(c, &a.confirmUninstall, "移到废纸篓？", a.uninstallSummary(), "取消", "卸载") == 1 {
		a.uninstallSelected()
	}
}

func mygoQuit() {
	// Kept as a named callback so keyboard and menu actions share one command.
	mygo.App.Quit()
}

func (a *application) panelHeader(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(54).Padding(9, 12).Gap(12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Box(c).Size(26, 26).Radius(6).Background(ui.Hex("#2563EB")).Center().Children(func() {
				ui.Icon(c, iconPulse).FontSize(15).TextColor(ui.RGB(255, 255, 255))
			})
			ui.Text(c, "Vitals").FontSize(15).Bold()
		})
		ui.Spacer(c)
		ui.Segmented(c, &a.page, "状态", "内存", "应用", "设置").
			Label("功能").
			Shrink(0)
	})
}

func (a *application) overview(c *ui.Context) {
	snapshot := a.snapshot
	title, detail, attention := healthSummary(snapshot, a.startupItems)
	accent := ui.Hex("#16A34A")
	if attention {
		accent = ui.Hex("#D97706")
	}
	section(c, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(32, 32).Radius(7).Background(accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, iconShield).FontSize(17).TextColor(accent)
			})
			ui.Column(c).MinWidth(0).Grow(1).Gap(2).Children(func() {
				ui.Text(c, title).FontSize(14).Bold()
				ui.Text(c, detail).FontSize(10).TextColor(c.Theme().TextMuted).SingleLine()
			})
		})
	}).Padding(12)
	ui.Grid(c).Columns(2).Gap(10).Children(func() {
		metricCard(c, "CPU", fmt.Sprintf("%.0f%%", snapshot.CPUPercent), "处理器", iconPulse, ui.Hex("#2563EB"), snapshot.HistoryCPU, 100)
		metricCard(c, "RAM", fmt.Sprintf("%.0f%%", snapshot.MemoryPercent), formatBytes(snapshot.MemoryUsed), iconMemory, ui.Hex("#0F9F8F"), snapshot.HistoryMemory, 100)
		metricCard(c, "GPU", fmt.Sprintf("%.0f%%", snapshot.GPUPercent), "图形负载", iconGPU, ui.Hex("#D97706"), nil, 100)
		metricCard(c, "网络", "↓ "+compactRate(snapshot.NetworkDown), "↑ "+compactRate(snapshot.NetworkUp), iconNetwork, ui.Hex("#DB2777"), snapshot.HistoryNetwork, maxHistory(snapshot.HistoryNetwork))
	})
	section(c, func() {
		t := c.Theme()
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(3).Grow(1).Children(func() {
				ui.Text(c, "Macintosh HD").FontSize(14).Bold()
				ui.Text(c, fmt.Sprintf("已使用 %s，共 %s", formatBytes(snapshot.DiskUsed), formatBytes(snapshot.DiskTotal))).FontSize(12).TextColor(t.TextMuted)
			})
			ui.Text(c, fmt.Sprintf("%.0f%%", snapshot.DiskPercent)).FontSize(16).FontWeight(600)
		})
		ui.Progress(c, snapshot.DiskPercent/100).Label("磁盘使用率").Height(5)
		if snapshot.Battery.Present {
			ui.Divider(c)
			batterySummary(c, snapshot.Battery)
		}
	}).Padding(14).Gap(8)
	a.startupHealth(c)
}

func (a *application) startupHealth(c *ui.Context) {
	t := c.Theme()
	services := c.Services()
	_, review := startupCounts(a.startupItems)
	accent := ui.Hex("#16A34A")
	detail := fmt.Sprintf("%d 个启动项 · 未发现明显异常", len(a.startupItems))
	if a.startupLoading {
		detail = "正在检查可读取的启动项…"
	} else if a.startupError != "" {
		accent = ui.Hex("#D97706")
		detail = a.startupError
	} else if review > 0 {
		accent = ui.Hex("#D97706")
		detail = fmt.Sprintf("%d 个启动项 · %d 个需关注", len(a.startupItems), review)
	}
	section(c, func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(30, 30).Radius(7).Background(accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, iconRocket).FontSize(16).TextColor(accent)
			})
			ui.Column(c).MinWidth(0).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "启动项健康").FontSize(13).Bold()
				ui.Text(c, detail).FontSize(10).TextColor(t.TextMuted).SingleLine()
			})
			ui.Button(c, "").
				Label("重新检查启动项").
				Tooltip("重新检查").
				Disabled(a.startupLoading).
				Children(func() {
					if a.startupLoading {
						ui.Spinner(c)
					} else {
						ui.Icon(c, iconRefresh).FontSize(15)
					}
				}).
				OnClick(a.refreshStartupItems)
			ui.Button(c, "系统管理").OnClick(func() { services.OpenURL(startupSettingsURL) })
		})
	}).Padding(12)
}

func batterySummary(c *ui.Context, battery core.BatteryInfo) {
	t := c.Theme()
	state := "使用电池"
	if battery.Charging {
		state = "正在充电"
	} else if battery.ExternalPower {
		state = "已连接电源"
	}
	health := "检测中"
	if battery.HealthPercent > 0 {
		health = fmt.Sprintf("健康度约 %d%%", battery.HealthPercent)
	}
	ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, iconBattery).FontSize(17).TextColor(ui.Hex("#16A34A"))
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Text(c, fmt.Sprintf("电池 %d%% · %s", battery.ChargePercent, state)).FontSize(12).FontWeight(600)
			ui.Text(c, fmt.Sprintf("%s · %d 次循环", health, battery.CycleCount)).FontSize(10).TextColor(t.TextMuted)
		})
		if battery.TemperatureC > 0 {
			ui.Text(c, fmt.Sprintf("%.0f°C", battery.TemperatureC)).FontSize(11).TextColor(t.TextMuted)
		}
	})
}

func metricCard(c *ui.Context, title, value, detail string, icon *ui.SVG, accent ui.Color, history []float64, ceiling float64) {
	t := c.Theme()
	ui.Column(c).Height(112).Padding(12).Gap(6).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(26, 26).Radius(6).Background(accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, icon).FontSize(15).TextColor(accent)
			})
			ui.Text(c, title).FontSize(12).Bold().TextColor(t.TextMuted)
		})
		ui.Text(c, value).FontSize(21).Bold().FontFeatures("tnum").SingleLine()
		ui.Row(c).AlignItems(ui.End).Children(func() {
			ui.Text(c, detail).FontSize(10).TextColor(t.TextMuted).Grow(1).SingleLine()
			sparkline(c, history, ceiling, accent).Size(48, 18)
		})
	})
}

func (a *application) memory(c *ui.Context) {
	t := c.Theme()
	snapshot := a.snapshot
	section(c, func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, "可用内存").FontSize(12).TextColor(t.TextMuted)
				ui.Text(c, formatBytes(snapshot.MemoryFree)).FontSize(32).Bold().FontFeatures("tnum")
			})
			ui.Text(c, fmt.Sprintf("%.0f%%", snapshot.MemoryPercent)).FontSize(24).Bold().FontFeatures("tnum").TextColor(t.TextMuted)
		})
		ui.Progress(c, snapshot.MemoryPercent/100).Label("内存使用率").Height(5)
		ui.Row(c).Children(func() {
			ui.Text(c, fmt.Sprintf("已用 %s", formatBytes(snapshot.MemoryUsed))).FontSize(12).TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("总计 %s", formatBytes(snapshot.MemoryTotal))).FontSize(12).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "内存构成").FontSize(14).Bold()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("交换区 %s / %s", formatBytes(snapshot.SwapUsed), formatBytes(snapshot.SwapTotal))).FontSize(11).TextColor(t.TextMuted)
		})
		memoryComposition(c, snapshot)
		ui.Grid(c).Columns(3).Gap(8).Children(func() {
			memoryLegend(c, "活跃", snapshot.MemoryActive, ui.Hex("#2563EB"))
			memoryLegend(c, "非活跃", snapshot.MemoryInactive, ui.Hex("#60A5FA"))
			memoryLegend(c, "联动", snapshot.MemoryWired, ui.Hex("#0F9F8F"))
			memoryLegend(c, "压缩", snapshot.MemoryCompressed, ui.Hex("#D97706"))
			memoryLegend(c, "空闲", snapshot.MemoryKernelFree, ui.Hex("#D1D5DB"))
			memoryLegend(c, "可用", snapshot.MemoryFree, ui.Hex("#16A34A"))
		})
	}).Padding(14).Gap(9)

	section(c, func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "按应用占用").FontSize(14).Bold()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("前 %d 项 · 常驻内存 RSS", len(snapshot.MemoryConsumers))).FontSize(11).TextColor(t.TextMuted)
		})
		if a.memoryResult != "" {
			ui.Text(c, a.memoryResult).FontSize(11).FontWeight(600).TextColor(ui.Hex("#16A34A"))
		}
		if len(snapshot.MemoryConsumers) == 0 {
			ui.Row(c).Gap(8).PaddingY(12).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在汇总进程内存…").FontSize(12).TextColor(t.TextMuted)
			})
		}
		for _, consumer := range snapshot.MemoryConsumers {
			consumer := consumer
			name := a.localizedApplicationName(consumer.AppPath, consumer.Name)
			ui.Column(c.Key("memory:" + consumer.AppPath + ":" + consumer.Name)).Gap(3).PaddingY(2).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					a.applicationImage(c, consumer.AppPath, name, 26)
					ui.Column(c).MinWidth(0).Grow(1).Gap(1).Children(func() {
						ui.Text(c, name).FontSize(12).FontWeight(600).SingleLine()
						ui.Text(c, fmt.Sprintf("%d 个进程", consumer.ProcessCount)).FontSize(10).TextColor(t.TextMuted)
					})
					ui.Column(c).AlignItems(ui.End).Gap(1).Children(func() {
						ui.Text(c, formatBytes(consumer.RSS)).FontSize(12).FontWeight(600).FontFeatures("tnum")
						ui.Text(c, fmt.Sprintf("%.1f%%", consumer.Percent)).FontSize(10).TextColor(t.TextMuted).FontFeatures("tnum")
					})
					if a.canReleaseApplicationMemory(consumer.AppPath) {
						label := "释放"
						if a.releasingApp == consumer.AppPath {
							label = "释放中…"
						}
						ui.Button(c, label).
							Label(name + "释放内存").
							Disabled(a.releasingApp != "").
							Tooltip("退出应用并释放其内存").
							OnClick(func() { a.requestApplicationMemoryRelease(consumer) })
					}
				})
				ui.Progress(c, min(consumer.Percent/100, 1)).Label(name + "内存占比").Height(3)
			})
		}
	}).Padding(14).Gap(8)
}

func (a *application) settingsView(c *ui.Context) {
	t := c.Theme()
	section(c, func() {
		ui.Text(c, "启动").FontSize(14).Bold()
		settingsRow(c, "登录时启动", "登录 macOS 后在菜单栏运行", func() {
			if ui.Switch(c, &a.settings.OpenAtLogin).Label("登录时启动").Changed() {
				a.setOpenAtLogin(a.settings.OpenAtLogin)
			}
		})
	}).Padding(14).Gap(10)

	section(c, func() {
		ui.Text(c, "状态栏").FontSize(14).Bold()
		a.metricSetting(c, "CPU", "处理器使用率", &a.settings.ShowCPU)
		a.metricSetting(c, "RAM", "内存使用率", &a.settings.ShowRAM)
		a.metricSetting(c, "GPU", "图形处理器使用率", &a.settings.ShowGPU)
		a.metricSetting(c, "网络", "实时下载与上传速率", &a.settings.ShowNetwork)
	}).Padding(14).Gap(8)

	section(c, func() {
		ui.Text(c, "健康提醒").FontSize(14).Bold()
		settingsRow(c, "持续异常提醒", "CPU 或内存持续高负载、磁盘空间不足时提醒", func() {
			if ui.Switch(c, &a.settings.Alerts).Label("持续异常提醒").Changed() {
				a.health.Reset()
				a.persistSettings()
			}
		})
		if a.settings.Alerts {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				status := a.notificationResult
				if status == "" {
					status = "首次发送时由 macOS 请求通知权限"
				}
				ui.Text(c, status).FontSize(10).TextColor(t.TextMuted).Grow(1)
				label := "测试提醒"
				if a.testingNotification {
					label = "发送中…"
				}
				ui.Button(c, label).Disabled(a.testingNotification).OnClick(a.testHealthNotification)
			})
		}
	}).Padding(14).Gap(10)

	section(c, func() {
		ui.Text(c, "更新与外观").FontSize(14).Bold()
		settingsRow(c, "刷新频率", "状态栏与系统数据的更新间隔", func() {
			if ui.Select(c, &a.settings.RefreshRate, []string{"1 秒", "2 秒", "5 秒"}).Label("刷新频率").Changed() {
				a.persistSettings()
			}
		})
		settingsRow(c, "外观", "自动跟随 macOS 深色或浅色模式", func() {
			ui.Text(c, "跟随系统").FontSize(11).TextColor(t.TextMuted)
		})
	}).Padding(14).Gap(10)

	section(c, func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "Vitals").FontSize(13).Bold()
				version := mygo.App.Version()
				if version == "" {
					version = "调试版本"
				} else {
					version = "版本 " + version
				}
				ui.Text(c, version+" · 数据仅保存在本机").FontSize(10).TextColor(t.TextMuted)
			})
			ui.Button(c, "退出 Vitals").OnClick(mygo.App.Quit)
		})
	}).Padding(14)
}

func startupCounts(items []core.StartupItem) (enabled, review int) {
	for _, item := range items {
		if item.Enabled {
			enabled++
		}
		if item.NeedsReview {
			review++
		}
	}
	return enabled, review
}

func (a *application) metricSetting(c *ui.Context, title, detail string, value *bool) {
	disabled := *value && a.settings.visibleMetricCount() == 1
	settingsRow(c, title, detail, func() {
		if ui.Switch(c, value).Label(title).Disabled(disabled).Changed() {
			a.persistSettings()
		}
	})
}

func settingsRow(c *ui.Context, title, detail string, control func()) {
	t := c.Theme()
	ui.Row(c).MinHeight(38).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, title).FontSize(12).FontWeight(600)
			ui.Text(c, detail).FontSize(10).TextColor(t.TextMuted).SingleLine()
		})
		control()
	})
}

func memoryComposition(c *ui.Context, snapshot core.Snapshot) {
	total := snapshot.MemoryTotal
	if total == 0 {
		total = 1
	}
	parts := []struct {
		value uint64
		color ui.Color
	}{
		{snapshot.MemoryActive, ui.Hex("#2563EB")},
		{snapshot.MemoryInactive, ui.Hex("#60A5FA")},
		{snapshot.MemoryWired, ui.Hex("#0F9F8F")},
		{snapshot.MemoryCompressed, ui.Hex("#D97706")},
		{snapshot.MemoryKernelFree, ui.Hex("#D1D5DB")},
	}
	ui.Row(c).Height(14).Radius(4).Clip().Background(c.Theme().Border).Children(func() {
		for _, part := range parts {
			width := float32(part.value) / float32(total) * 100
			if width <= 0 {
				continue
			}
			ui.Box(c).WidthPercent(min(width, 100)).Height(14).Shrink(0).Background(part.color)
		}
	})
}

func memoryLegend(c *ui.Context, label string, value uint64, color ui.Color) {
	t := c.Theme()
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(8, 8).Radius(2).Background(color).Shrink(0)
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, formatBytes(value)).FontSize(11).FontWeight(600).FontFeatures("tnum")
	})
}

func (a *application) localizedApplicationName(appPath, fallback string) string {
	for index := range a.apps {
		if a.apps[index].Path == appPath {
			return a.apps[index].Name
		}
	}
	return fallback
}

func (a *application) applicationImage(c *ui.Context, appPath, name string, size float32) {
	if icon := a.appIcons[appPath]; icon != nil {
		ui.Image(c, icon).Size(size, size).Fit(ui.Contain).Radius(6).Label(name)
		return
	}
	ui.Avatar(c, name, nil).Size(size, size)
}

func (a *application) uninstaller(c *ui.Context) {
	if a.selectedApp >= 0 && a.selectedApp < len(a.apps) {
		a.uninstallerDetail(c)
		return
	}
	a.uninstallerList(c)
}

func (a *application) uninstallerList(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().MinHeight(0).Gap(10).Children(func() {
		ui.Text(c, "应用管理").FontSize(15).Bold()
		ui.TextInput(c, &a.appQuery).Label("搜索应用").Placeholder("搜索应用")
		if a.loadingApps {
			ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
				ui.Spinner(c)
				ui.Text(c, "正在读取应用…").FontSize(12).TextColor(t.TextMuted)
			})
			return
		}
		indices := filteredApplications(a.apps, a.appQuery)
		ui.Scroll(c).Grow(1).Gap(3).Label("应用列表").Children(func() {
			for _, index := range indices {
				index := index
				app := a.apps[index]
				row := ui.Row(c.Key(index)).Height(50).Padding(7, 9).Gap(10).Radius(7).AlignItems(ui.Center).Cursor(ui.CursorPointer)
				if row.Hovered() {
					row.Background(t.Surface)
				}
				row.OnClick(func() { a.selectApplication(index) }).Children(func() {
					a.applicationImage(c, app.Path, app.Name, 30)
					ui.Column(c).MinWidth(0).Grow(1).Gap(1).Children(func() {
						ui.Text(c, app.Name).FontSize(13).FontWeight(600).SingleLine()
						detail := app.Version
						if app.Running {
							detail = "正在运行 · " + detail
						}
						ui.Text(c, detail).FontSize(10).TextColor(t.TextMuted).SingleLine()
					})
				})
			}
		})
		ui.Text(c, fmt.Sprintf("%d 个应用", len(indices))).FontSize(11).TextColor(t.TextMuted)
	})
}

func (a *application) uninstallerDetail(c *ui.Context) {
	t := c.Theme()
	services := c.Services()
	app := &a.apps[a.selectedApp]
	memory, processes := a.applicationMemory(app.Path)
	startupEnabled, startupTotal := a.applicationStartupCounts(*app)
	ui.Column(c).Fill().MinHeight(0).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Button(c, "").Label("返回应用列表").Tooltip("返回应用列表").Children(func() {
				ui.Icon(c, iconBack).FontSize(17)
			}).OnClick(func() {
				a.selectedApp = -1
			})
			ui.Text(c, "应用健康档案").FontSize(15).Bold()
		})
		ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(10, 12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
			a.applicationImage(c, app.Path, app.Name, 42)
			ui.Column(c).MinWidth(0).Grow(1).Gap(2).Children(func() {
				ui.Text(c, app.Name).FontSize(17).Bold().SingleLine()
				ui.Text(c, app.BundleID).FontSize(10).TextColor(t.TextMuted).SingleLine()
			})
			if app.Running {
				label := "退出应用"
				if a.quittingApp {
					label = "正在退出…"
				}
				ui.Button(c, label).Disabled(a.quittingApp).OnClick(a.quitSelectedApplication)
			}
		})
		ui.Grid(c).Columns(3).Gap(8).Children(func() {
			healthMetric(c, "常驻内存", formatBytes(memory), fmt.Sprintf("%d 个进程", processes))
			healthMetric(c, "启动项", fmt.Sprint(startupEnabled), fmt.Sprintf("共 %d 项", startupTotal))
			storage := app.TotalBytes
			if storage == 0 {
				storage = app.Size
			}
			healthMetric(c, "磁盘占用", formatBytes(storage), "应用与已选关联项")
		})
		if startupTotal > 0 {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("该应用关联 %d 个启动项，其中 %d 个已启用", startupTotal, startupEnabled)).
					FontSize(10).
					TextColor(t.TextMuted).
					Grow(1)
				ui.Button(c, "系统管理").OnClick(func() { services.OpenURL(startupSettingsURL) })
			})
		} else if a.startupLoading {
			ui.Text(c, "正在关联启动项…").FontSize(10).TextColor(t.TextMuted)
		}
		ui.Row(c).Children(func() {
			ui.Text(c, "将移到废纸篓").FontSize(13).Bold()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d 项 · %s", selectedFileCount(*app), formatBytes(selectedBytes(*app)))).FontSize(11).TextColor(t.TextMuted)
		})
		ui.Scroll(c).Grow(1).Gap(6).Padding(10, 12).Radius(8).Background(t.Surface).Border(1, t.Border).Label("关联文件列表").Children(func() {
			locked := true
			ui.Checkbox(c, &locked, app.Path).Disabled(true).Label("应用本体")
			if a.scanningRelated && !app.Scanned {
				ui.Row(c).Gap(8).PaddingY(8).Children(func() {
					ui.Spinner(c)
					ui.Text(c, "正在扫描关联文件…").FontSize(12).TextColor(t.TextMuted)
				})
			}
			for index := range app.Related {
				file := &app.Related[index]
				ui.Row(c.Key(file.Path)).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Checkbox(c, &file.Selected, file.Path).Grow(1)
					ui.Text(c, file.Confidence+"置信度").FontSize(10).TextColor(t.TextMuted)
					ui.Text(c, formatBytes(file.Size)).Width(58).TextAlign(ui.End).FontSize(10).TextColor(t.TextMuted)
				})
			}
			if app.Scanned && len(app.Related) == 0 {
				ui.Text(c, "未发现额外关联文件。").FontSize(12).TextColor(t.TextMuted).PaddingY(8)
			}
		})
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "移除后可从废纸篓恢复").FontSize(11).TextColor(t.TextMuted).Grow(1)
			label := "卸载"
			if a.uninstalling {
				label = "正在移动…"
			}
			ui.PrimaryButton(c, label).Disabled(app.Running || !app.Scanned || a.uninstalling).OnClick(func() {
				a.confirmUninstall = true
			})
		})
	})
}

func healthMetric(c *ui.Context, label, value, detail string) {
	t := c.Theme()
	ui.Column(c).Padding(8).Gap(1).Radius(7).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, value).FontSize(13).Bold().FontFeatures("tnum").SingleLine()
		ui.Text(c, label).FontSize(10).FontWeight(600)
		ui.Text(c, detail).FontSize(9).TextColor(t.TextMuted).SingleLine()
	})
}

func (a *application) applicationMemory(path string) (uint64, int) {
	for _, consumer := range a.snapshot.MemoryConsumers {
		if consumer.AppPath == path {
			return consumer.RSS, consumer.ProcessCount
		}
	}
	return 0, 0
}

func (a *application) applicationStartupCounts(app core.AppInfo) (enabled, total int) {
	for _, item := range a.startupItems {
		matches := item.AppPath == app.Path ||
			app.BundleID != "" && (item.BundleID == app.BundleID || strings.HasPrefix(item.Label, app.BundleID+"."))
		if !matches {
			continue
		}
		total++
		if item.Enabled {
			enabled++
		}
	}
	return enabled, total
}

func section(c *ui.Context, body func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(18).Gap(14).Radius(8).Background(t.Surface).Border(1, t.Border).Children(body)
}

func sparkline(c *ui.Context, values []float64, ceiling float64, accent ui.Color) ui.Element {
	t := c.Theme()
	return ui.Box(c).MinWidth(40).Background(t.Border.Alpha(0.16)).Radius(5).Draw(func(p *ui.Painter, rect ui.Rect) {
		if len(values) < 2 || ceiling <= 0 {
			return
		}
		var path ui.Path
		for index, value := range values {
			x := rect.X + 4 + float32(index)/float32(len(values)-1)*(rect.W-8)
			normalized := float32(math.Min(1, math.Max(0, value/ceiling)))
			y := rect.Y + rect.H - 4 - normalized*(rect.H-8)
			if index == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
		p.StrokePath(&path, 2, accent)
	})
}

func maxHistory(values []float64) float64 {
	maximum := 1.0
	for _, value := range values {
		maximum = max(maximum, value)
	}
	return maximum
}

func selectedFileCount(app core.AppInfo) int {
	count := 1
	for _, file := range app.Related {
		if file.Selected {
			count++
		}
	}
	return count
}

func selectedBytes(app core.AppInfo) uint64 {
	total := app.Size
	for _, file := range app.Related {
		if file.Selected {
			total += file.Size
		}
	}
	return total
}

func (a *application) uninstallSummary() string {
	if a.selectedApp < 0 || a.selectedApp >= len(a.apps) {
		return ""
	}
	app := a.apps[a.selectedApp]
	return fmt.Sprintf("%s 及 %d 个已选关联项将移到废纸篓，可在清空废纸篓前恢复。", app.Name, selectedFileCount(app)-1)
}

func formatBytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor, exponent := uint64(unit), 0
	for quotient := value / unit; quotient >= unit && exponent < 4; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}

func formatRate(value uint64) string {
	text := formatBytes(value)
	return strings.Replace(text, "iB", "B", 1) + "/s"
}
