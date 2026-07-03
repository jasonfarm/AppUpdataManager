package main

import (
	_ "embed"
	"example.com/appupdatemanager/client/internal/autostart"
	"example.com/appupdatemanager/client/internal/config"
	"example.com/appupdatemanager/client/internal/server"
	"example.com/appupdatemanager/client/internal/software"
	"example.com/appupdatemanager/client/internal/sysinfo"
	"example.com/appupdatemanager/client/internal/systray"
	"example.com/appupdatemanager/client/internal/updater"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const appName = "appUpdateManagerClient"

// iconSVGBytes 是内嵌的 SVG 图标数据，用于设置应用图标。
//
//go:embed assets/icon.svg
var iconSVGBytes []byte

// iconICOBytes 是内嵌的 ICO 图标数据，用于设置系统托盘图标。
//
//go:embed assets/icon.ico
var iconICOBytes []byte

// main 是客户端应用的入口函数，负责加载配置、构建 UI、连接服务器并运行主循环。
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config error: %v\n", err)
		cfg = config.Default()
	}

	a := app.NewWithID("com.appupdatemanager.client")
	appIcon := fyne.NewStaticResource("icon.svg", iconSVGBytes)
	a.SetIcon(appIcon)
	trayIcon := fyne.NewStaticResource("icon.ico", iconICOBytes)
	w := a.NewWindow("AppUpdateManager 客户端")
	w.Resize(fyne.NewSize(600, 500))

	sysInfo, err := sysinfo.Collect()
	if err != nil {
		sysInfo = &sysinfo.Info{}
	}

	swManager := software.NewManager(filepath.Join(config.Dir(), "software"))

	// Server client
	var srvClient *server.Client
	commandHandler := func(cmd string, payload map[string]string) {
		handleCommand(cfg, swManager, cmd, payload)
	}
	srvClient = server.NewClient(cfg, commandHandler)

	// UI
	statusLabel := widget.NewLabel(fmt.Sprintf("状态: 未连接 | 软件版本: %s | 运行中: %v", swManager.CurrentVersion(), swManager.IsRunning()))
	serverHostEntry := widget.NewEntry()
	serverHostEntry.SetText(cfg.ServerHost)
	serverPortEntry := widget.NewEntry()
	serverPortEntry.SetText(cfg.ServerPort)
	clientNameEntry := widget.NewEntry()
	clientNameEntry.SetText(cfg.ClientName)
	versionEntry := widget.NewEntry()
	versionEntry.SetText(cfg.ClientVersion)
	versionEntry.Disable()
	autoStartCheck := widget.NewCheck("开机自启动", nil)
	autoStartCheck.SetChecked(autostart.IsEnabled(appName))

	saveBtn := widget.NewButton("保存设置", func() {
		cfg.ServerHost = serverHostEntry.Text
		cfg.ServerPort = serverPortEntry.Text
		cfg.ClientName = clientNameEntry.Text
		cfg.AutoStart = autoStartCheck.Checked
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if cfg.AutoStart {
			exe, _ := os.Executable()
			_ = autostart.Enable(appName, exe)
		} else {
			_ = autostart.Disable(appName)
		}
		dialog.ShowInformation("保存成功", "设置已保存", w)
	})

	connectBtn := widget.NewButton("连接服务器", func() {
		if srvClient != nil {
			_ = srvClient.Close()
		}
		srvClient = server.NewClient(cfg, commandHandler)
		hb := &server.HeartbeatData{
			IP:        sysInfo.IP,
			OSVersion: sysInfo.OSVersion,
			Memory:    sysInfo.Memory,
			CPU:       sysInfo.CPU,
		}
		if err := srvClient.Connect(hb); err != nil {
			dialog.ShowError(err, w)
			return
		}
		statusLabel.SetText(fmt.Sprintf("状态: 已连接 | 软件版本: %s | 运行中: %v", swManager.CurrentVersion(), swManager.IsRunning()))
	})

	// Tabs
	settingsTab := container.NewVBox(
		widget.NewCard("服务器设置", "", container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("服务器地址", serverHostEntry),
				widget.NewFormItem("端口", serverPortEntry),
			),
		)),
		widget.NewCard("客户端设置", "", container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("客户端名称", clientNameEntry),
				widget.NewFormItem("客户端版本", versionEntry),
			),
			autoStartCheck,
		)),
		container.NewHBox(saveBtn, connectBtn),
	)

	statusTab := container.NewVBox(
		statusLabel,
		widget.NewLabel(fmt.Sprintf("本机 IP: %s", sysInfo.IP)),
		widget.NewLabel(fmt.Sprintf("操作系统: %s", sysInfo.OSVersion)),
		widget.NewLabel(fmt.Sprintf("内存: %s", sysInfo.Memory)),
		widget.NewLabel(fmt.Sprintf("CPU: %s", sysInfo.CPU)),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("设置", settingsTab),
		container.NewTabItem("状态", statusTab),
	)

	w.SetContent(tabs)

	// 系统托盘初始化
	systray.Setup(a, trayIcon, func() {
		w.Show()
	}, func() {
		if srvClient != nil {
			_ = srvClient.Close()
		}
		a.Quit()
	})

	// Minimize to tray on close
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// Status updater
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			statusLabel.SetText(fmt.Sprintf("状态: %s | 软件版本: %s | 运行中: %v | 运行时长: %ds",
				connectionStatus(srvClient),
				swManager.CurrentVersion(),
				swManager.IsRunning(),
				swManager.Runtime(),
			))
			if srvClient != nil {
				srvClient.SetStatus(swManager.CurrentVersion(), swManager.IsRunning(), swManager.Runtime())
			}
		}
	}()

	// Auto connect
	go func() {
		time.Sleep(1 * time.Second)
		hb := &server.HeartbeatData{
			IP:        sysInfo.IP,
			OSVersion: sysInfo.OSVersion,
			Memory:    sysInfo.Memory,
			CPU:       sysInfo.CPU,
		}
		_ = srvClient.Connect(hb)
	}()

	w.ShowAndRun()
}

// connectionStatus 根据客户端连接状态返回对应的中文状态描述字符串。
func connectionStatus(c *server.Client) string {
	if c == nil {
		return "未连接"
	}
	return "已连接"
}

// handleCommand 处理服务器下发的控制命令，包括软件更新、客户端自更新、启动、停止和重启操作。
func handleCommand(cfg *config.Config, mgr *software.Manager, cmd string, payload map[string]string) {
	switch cmd {
	case "update_software":
		version := payload["version"]
		downloadURL := payload["download_url"]
		filename := filepath.Base(downloadURL)
		if filename == "" {
			filename = "app.exe"
		}
		dir, err := mgr.EnsureVersion(version)
		if err != nil {
			return
		}
		savePath := filepath.Join(dir, filename)
		if err := server.DownloadFile(cfg.ServerURL(), downloadURL, savePath); err != nil {
			return
		}
		_ = mgr.Stop()
		_ = mgr.Start(version, filename)
	case "update_self":
		downloadURL := payload["download_url"]
		savePath, err := updater.DownloadPath()
		if err != nil {
			return
		}
		if err := server.DownloadFile(cfg.ServerURL(), downloadURL, savePath); err != nil {
			return
		}
		_ = updater.SelfUpdate(savePath)
	case "start":
		latestVer := mgr.CurrentVersion()
		if latestVer == "" {
			return
		}
		// Find executable in current version directory
		dir := filepath.Join(mgr.VersionsDir(), latestVer)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			return
		}
		_ = mgr.Start(latestVer, entries[0].Name())
	case "stop":
		_ = mgr.Stop()
	case "restart":
		latestVer := mgr.CurrentVersion()
		if latestVer == "" {
			return
		}
		dir := filepath.Join(mgr.VersionsDir(), latestVer)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			return
		}
		_ = mgr.Restart(latestVer, entries[0].Name())
	}
}
