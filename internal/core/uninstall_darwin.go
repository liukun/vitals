//go:build darwin

package core

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"howett.net/plist"
)

type bundleMetadata struct {
	Name       string `plist:"CFBundleDisplayName"`
	BundleName string `plist:"CFBundleName"`
	Identifier string `plist:"CFBundleIdentifier"`
	Version    string `plist:"CFBundleShortVersionString"`
	IconFile   string `plist:"CFBundleIconFile"`
	IconName   string `plist:"CFBundleIconName"`
}

// FullDiskAccessError 表示目标受 macOS 隐私保护，需要用户授予完全磁盘访问权限。
type FullDiskAccessError struct {
	Path string
}

func (e *FullDiskAccessError) Error() string {
	return fmt.Sprintf(
		"无法移除受 macOS 保护的数据：\n%s\n\n请允许 Vitals 完全磁盘访问后重试。应用本体尚未移除",
		e.Path,
	)
}

// ListApplications 返回用户可管理的应用，不包含 macOS 系统应用。
func ListApplications(ctx context.Context) ([]AppInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	roots := []string{"/Applications", filepath.Join(home, "Applications")}
	var apps []AppInfo
	for _, root := range roots {
		found, walkErr := applicationsIn(root, 2)
		if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
			return nil, walkErr
		}
		for _, path := range found {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			meta, readErr := readBundleMetadata(path)
			if readErr != nil {
				continue
			}
			fallbackName := firstNonEmpty(meta.Name, meta.BundleName, strings.TrimSuffix(filepath.Base(path), ".app"))
			name := localizedApplicationName(ctx, path, fallbackName)
			apps = append(apps, AppInfo{
				Name:     name,
				Path:     path,
				BundleID: meta.Identifier,
				Version:  meta.Version,
				Size:     pathSize(path),
				Running:  ApplicationRunning(ctx, path),
				Selected: true,
				IconPNG:  applicationIconPNG(ctx, path, meta),
			})
		}
	}
	slices.SortFunc(apps, func(a, b AppInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return apps, nil
}

func localizedApplicationName(ctx context.Context, appPath, fallback string) string {
	output, err := exec.CommandContext(ctx, "/usr/bin/mdls", "-raw", "-name", "kMDItemDisplayName", appPath).Output()
	if err != nil {
		return fallback
	}
	name := strings.Trim(strings.TrimSpace(string(output)), `"`)
	name = strings.TrimSuffix(name, ".app")
	if name == "" || name == "(null)" {
		return fallback
	}
	return name
}

func applicationIconPNG(ctx context.Context, appPath string, meta bundleMetadata) []byte {
	resources := filepath.Join(appPath, "Contents", "Resources")
	var candidates []string
	for _, name := range []string{meta.IconFile, meta.IconName} {
		if name == "" {
			continue
		}
		if filepath.Ext(name) == "" {
			name += ".icns"
		}
		candidates = append(candidates, filepath.Join(resources, name))
	}
	if matches, _ := filepath.Glob(filepath.Join(resources, "*.icns")); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	iconPath := ""
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			iconPath = candidate
			break
		}
	}
	if iconPath == "" {
		return nil
	}
	temp, err := os.CreateTemp("", "vitals-app-icon-*.png")
	if err != nil {
		return nil
	}
	outputPath := temp.Name()
	_ = temp.Close()
	_ = os.Remove(outputPath)
	defer os.Remove(outputPath)
	command := exec.CommandContext(ctx, "/usr/bin/sips", "-s", "format", "png", "-z", "128", "128", iconPath, "--out", outputPath)
	if command.Run() != nil {
		return nil
	}
	data, _ := os.ReadFile(outputPath)
	return data
}

func applicationsIn(root string, depth int) ([]string, error) {
	var apps []string
	var visit func(string, int) error
	visit = func(dir string, level int) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
				apps = append(apps, path)
				continue
			}
			if level < depth {
				_ = visit(path, level+1)
			}
		}
		return nil
	}
	return apps, visit(root, 0)
}

func readBundleMetadata(appPath string) (bundleMetadata, error) {
	data, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return bundleMetadata{}, err
	}
	var meta bundleMetadata
	if _, err := plist.Unmarshal(data, &meta); err != nil {
		return bundleMetadata{}, err
	}
	return meta, nil
}

// ApplicationRunning 判断应用包当前是否仍有主程序或辅助进程。
func ApplicationRunning(ctx context.Context, appPath string) bool {
	command := exec.CommandContext(ctx, "/usr/bin/pgrep", "-f", appPath+"/Contents/MacOS/")
	return command.Run() == nil
}

// ScanRelatedFiles 根据 bundle id 与应用名扫描关联文件，并标注命中置信度。
func ScanRelatedFiles(ctx context.Context, app AppInfo) ([]RelatedFile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	roots := []string{
		filepath.Join(home, "Library", "Application Support"),
		filepath.Join(home, "Library", "Caches"),
		filepath.Join(home, "Library", "Preferences"),
		filepath.Join(home, "Library", "Logs"),
		filepath.Join(home, "Library", "Saved Application State"),
		filepath.Join(home, "Library", "Containers"),
		filepath.Join(home, "Library", "Group Containers"),
		filepath.Join(home, "Library", "LaunchAgents"),
		"/Library/Application Support",
		"/Library/LaunchAgents",
		"/Library/LaunchDaemons",
		"/Library/PrivilegedHelperTools",
	}
	var files []RelatedFile
	for _, root := range roots {
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			confidence, matched := relatedConfidence(entry.Name(), app)
			if !matched {
				continue
			}
			path := filepath.Join(root, entry.Name())
			files = append(files, RelatedFile{
				Path:       path,
				Size:       pathSize(path),
				Kind:       filepath.Base(root),
				Confidence: confidence,
				Selected:   confidence == "高",
			})
		}
	}
	slices.SortFunc(files, func(a, b RelatedFile) int {
		if a.Confidence != b.Confidence {
			return strings.Compare(a.Confidence, b.Confidence)
		}
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}

func relatedConfidence(filename string, app AppInfo) (string, bool) {
	value := strings.ToLower(filename)
	bundleID := strings.ToLower(strings.TrimSpace(app.BundleID))
	if bundleID != "" && (value == bundleID || strings.HasPrefix(value, bundleID+".") || strings.HasPrefix(value, bundleID+"-")) {
		return "高", true
	}
	name := strings.ToLower(strings.TrimSpace(app.Name))
	if len(name) >= 4 && (value == name || strings.HasPrefix(value, name+".") || strings.HasPrefix(value, name+" ")) {
		return "中", true
	}
	return "", false
}

// QuitApplication 请求目标应用正常退出，不强制终止进程。
func QuitApplication(ctx context.Context, app AppInfo) error {
	if strings.TrimSpace(app.BundleID) == "" {
		return fmt.Errorf("应用缺少 bundle id，无法安全退出")
	}
	script := `on run argv
tell application id (item 1 of argv) to quit
end run`
	if output, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", script, app.BundleID).CombinedOutput(); err != nil {
		return fmt.Errorf("退出 %s: %s", app.Name, strings.TrimSpace(string(output)))
	}
	return nil
}

// MoveToTrash 按给定顺序将路径移入废纸篓，遇到失败立即停止。
// 调用方应将应用本体放在最后，确保关联文件失败时应用仍可用于重试。
func MoveToTrash(ctx context.Context, paths []string, trash func(string) error) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == "" || path == "/" || path == home {
			return fmt.Errorf("拒绝处理不安全路径 %q", path)
		}
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("读取 %s: %w", path, err)
		}
		if err := trash(path); err != nil {
			if protectedLibraryPath(path, home) {
				return &FullDiskAccessError{Path: path}
			}
			return fmt.Errorf("无法将以下项目移入废纸篓：\n%s\n\n%s", path, err)
		}
	}
	return nil
}

func protectedLibraryPath(path, home string) bool {
	protected := []string{
		filepath.Join(home, "Library", "Containers"),
		filepath.Join(home, "Library", "Group Containers"),
		"/Library/LaunchDaemons",
		"/Library/PrivilegedHelperTools",
	}
	clean := filepath.Clean(path)
	for _, root := range protected {
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func pathSize(path string) uint64 {
	var total uint64
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			if info, statErr := entry.Info(); statErr == nil && info.Size() > 0 {
				total += uint64(info.Size())
			}
		}
		return nil
	})
	return total
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
