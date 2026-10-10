# Vitals

基于 [MyGo](https://github.com/egoist/mygo) 的 macOS 13+ 原生菜单栏健康工具：

1. 在菜单栏实时显示 CPU、内存、GPU 与网络速率。
2. 展示应用内存占用，并允许确认后退出单个应用释放内存。
3. 将应用内存、启动项和磁盘占用聚合为应用健康档案。
4. 扫描应用本体和关联文件，确认后统一移入废纸篓。
5. 只读体检可访问的第三方 LaunchAgent 和 LaunchDaemon，在状态页展示健康摘要，并跳转 macOS 管理启动项。
6. 在 MacBook 上显示电量、充电状态、估算健康度和循环次数。
7. 可选监测持续 30 秒以上的 CPU、内存和磁盘异常，并通过系统通知提醒。

## 开发

需要 Go 1.27.1 或更高版本。

```bash
go tool mygo dev
```

## 验证与构建

```bash
GOFLAGS=-buildvcs=false go test ./...
GOFLAGS=-buildvcs=false go tool mygo vet .
GOFLAGS=-buildvcs=false go tool mygo build
```

构建产物位于 `build/darwin-arm64/`。

## 安全边界

- 单应用内存释放会正常退出目标应用，未保存内容可能丢失。
- 卸载候选项只根据 bundle id 与应用名匹配；高置信度项目默认选中，中置信度项目由用户决定。
- 卸载不会永久删除文件，所有确认项都会移入废纸篓。
- 启动项扫描完全在本机执行，不上传配置、路径、签名或可执行文件。
- Vitals 不修改、停用或删除启动项，也不调用需要提权的现代后台任务查询。
- 启动项管理统一跳转到 macOS“登录项与扩展”设置处理。
- 健康提醒默认关闭，启用后首次发送通知时由 macOS 请求通知权限。
