package sysinfo

import (
	"fmt"
	"net"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// Info 保存本机的系统信息，包括 IP、操作系统、内存和 CPU。
type Info struct {
	IP        string `json:"ip"`         // IP 是本机非回环 IPv4 地址。
	OSVersion string `json:"os_version"` // OSVersion 是操作系统版本描述。
	Memory    string `json:"memory"`     // Memory 是内存使用情况的字符串描述。
	CPU       string `json:"cpu"`        // CPU 是 CPU 型号信息。
}

// Collect 收集本机的 IP、操作系统、内存和 CPU 信息并返回。
func Collect() (*Info, error) {
	info := &Info{}

	// Get outbound IP
	info.IP = getOutboundIP()

	// OS info
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	info.OSVersion = fmt.Sprintf("%s %s (%s)", hostInfo.Platform, hostInfo.PlatformVersion, runtime.GOARCH)

	// Memory
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	info.Memory = fmt.Sprintf("%.1fGB / %.1fGB", float64(vmStat.Used)/1024/1024/1024, float64(vmStat.Total)/1024/1024/1024)

	// CPU
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	if len(cpuInfo) > 0 {
		info.CPU = cpuInfo[0].ModelName
	}

	return info, nil
}

// getOutboundIP 获取本机第一个非回环的 IPv4 地址，获取失败返回空字符串。
func getOutboundIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
