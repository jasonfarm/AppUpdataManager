package main

import (
	_ "embed"
	"example.com/appupdatemanager/client/internal/autostart"
	"example.com/appupdatemanager/client/internal/config"
	"example.com/appupdatemanager/client/internal/logger"
	"example.com/appupdatemanager/client/internal/server"
	"example.com/appupdatemanager/client/internal/software"
	"example.com/appupdatemanager/client/internal/sysinfo"
	"example.com/appupdatemanager/client/internal/systray"
	"example.com/appupdatemanager/client/internal/updater"
	"example.com/appupdatemanager/client/internal/winutil"
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

// windowTitle 是客户端主窗口标题，也用于查找已有窗口。
const windowTitle = "AppUpdateManager 客户端"

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
	// 单实例检查：若已有实例运行，则向其窗口发送显示消息并退出当前进程。
	if !winutil.EnsureSingleInstance(appName, windowTitle) {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config error: %v\n", err)
		cfg = config.Default()
	}

	// 初始化日志记录器，日志写入用户配置目录下的 client.log 文件。
	logFilePath := filepath.Join(config.Dir(), "client.log")
	appLogger, err := logger.New(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger error: %v\n", err)
		appLogger = nil
	} else {
		defer appLogger.Close()
		appLogger.Infof("客户端启动，日志文件: %s", logFilePath)
	}

	a := app.NewWithID("com.appupdatemanager.client")
	appIcon := fyne.NewStaticResource("icon.svg", iconSVGBytes)
	a.SetIcon(appIcon)
	trayIcon := fyne.NewStaticResource("icon.ico", iconICOBytes)
	var mainWindow fyne.Window
	w := a.NewWindow(windowTitle)
	mainWindow = w
	w.Resize(fyne.NewSize(600, 500))

	// 注册显示窗口回调并子类化 WNDPROC，让重复运行实例可以通过窗口消息
	// 在主消息线程中触发 w.Show()，避免空白窗口。
	winutil.RegisterShowWindowCallback(func() {
		mainWindow.Show()
	})
	go func() {
		for i := 0; i < 200; i++ {
			if err := winutil.SubclassWindow(windowTitle); err == nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// 应用启动后将窗口居中显示
	a.Lifecycle().SetOnStarted(func() {
		winutil.CenterWindow(windowTitle)
	})

	sysInfo, err := sysinfo.Collect()
	if err != nil {
		if appLogger != nil {
			appLogger.Errorf("收集系统信息失败: %v", err)
		}
		sysInfo = &sysinfo.Info{}
	}

	swManager := software.NewManager(filepath.Join(config.Dir(), "software"))

	// Server client
	var srvClient *server.Client
	commandHandler := func(cmd string, payload map[string]string) {
		handleCommand(cfg, swManager, appLogger, cmd, payload)
	}
	srvClient = server.NewClient(cfg, appLogger, commandHandler)

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
			if appLogger != nil {
				appLogger.Errorf("保存配置失败: %v", err)
			}
			dialog.ShowError(err, w)
			return
		}
		if cfg.AutoStart {
			exe, _ := os.Executable()
			_ = autostart.Enable(appName, exe)
		} else {
			_ = autostart.Disable(appName)
		}
		if appLogger != nil {
			appLogger.Info("配置已保存")
		}
		dialog.ShowInformation("保存成功", "设置已保存", w)
	})

	connectBtn := widget.NewButton("连接服务器", func() {
		if srvClient != nil {
			_ = srvClient.Close()
		}
		srvClient = server.NewClient(cfg, appLogger, commandHandler)
		hb := &server.HeartbeatData{
			IP:        sysInfo.IP,
			OSVersion: sysInfo.OSVersion,
			Memory:    sysInfo.Memory,
			CPU:       sysInfo.CPU,
		}
		if err := srvClient.Connect(hb); err != nil {
			if appLogger != nil {
				appLogger.Errorf("手动连接服务器失败: %v", err)
			}
			dialog.ShowError(err, w)
			return
		}
		if appLogger != nil {
			appLogger.Info("手动连接服务器成功")
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

	var logTab fyne.CanvasObject
	if appLogger != nil {
		logTab = appLogger.CreateView()
	} else {
		logTab = container.NewCenter(widget.NewLabel("日志初始化失败"))
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("设置", settingsTab),
		container.NewTabItem("状态", statusTab),
		container.NewTabItem("日志", logTab),
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
		if err := srvClient.Connect(hb); err != nil {
			if appLogger != nil {
				appLogger.Errorf("自动连接服务器失败: %v", err)
			}
		}
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
func handleCommand(cfg *config.Config, mgr *software.Manager, log *logger.Logger, cmd string, payload map[string]string) {
	if log != nil {
		log.Infof("开始执行服务器命令: %s, 参数: %v", cmd, payload)
	}

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
			if log != nil {
				log.Errorf("准备软件版本目录失败: %v", err)
			}
			return
		}
		savePath := filepath.Join(dir, filename)
		if err := server.DownloadFile(log, cfg.ServerURL(), downloadURL, savePath); err != nil {
			return
		}
		_ = mgr.Stop()
		if err := mgr.Start(version, filename); err != nil {
			if log != nil {
				log.Errorf("启动新版本软件失败: %v", err)
			}
			return
		}
		if log != nil {
			log.Infof("软件已更新到版本 %s 并启动", version)
		}
	case "update_self":
		downloadURL := payload["download_url"]
		savePath, err := updater.DownloadPath()
		if err != nil {
			if log != nil {
				log.Errorf("获取自更新下载路径失败: %v", err)
			}
			return
		}
		if err := server.DownloadFile(log, cfg.ServerURL(), downloadURL, savePath); err != nil {
			return
		}
		if log != nil {
			log.Info("客户端自更新文件下载完成，准备退出并替换")
		}
		_ = updater.SelfUpdate(savePath)
	case "start":
		latestVer := mgr.CurrentVersion()
		if latestVer == "" {
			if log != nil {
				log.Warn("启动命令被忽略，当前没有可用软件版本")
			}
			return
		}
		// Find executable in current version directory
		dir := filepath.Join(mgr.VersionsDir(), latestVer)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			if log != nil {
				log.Errorf("查找可执行文件失败: %v", err)
			}
			return
		}
		if err := mgr.Start(latestVer, entries[0].Name()); err != nil {
			if log != nil {
				log.Errorf("启动软件失败: %v", err)
			}
			return
		}
		if log != nil {
			log.Info("软件已启动")
		}
	case "stop":
		if err := mgr.Stop(); err != nil {
			if log != nil {
				log.Errorf("停止软件失败: %v", err)
			}
			return
		}
		if log != nil {
			log.Info("软件已停止")
		}
	case "restart":
		latestVer := mgr.CurrentVersion()
		if latestVer == "" {
			if log != nil {
				log.Warn("重启命令被忽略，当前没有可用软件版本")
			}
			return
		}
		dir := filepath.Join(mgr.VersionsDir(), latestVer)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			if log != nil {
				log.Errorf("查找可执行文件失败: %v", err)
			}
			return
		}
		if err := mgr.Restart(latestVer, entries[0].Name()); err != nil {
			if log != nil {
				log.Errorf("重启软件失败: %v", err)
			}
			return
		}
		if log != nil {
			log.Info("软件已重启")
		}
	default:
		if log != nil {
			log.Warnf("未知的服务器命令: %s", cmd)
		}
	}
}
