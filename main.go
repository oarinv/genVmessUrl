package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type User struct {
	UUID    string `json:"uuid"`
	AlterID int    `json:"alterId"`
}

type Inbound struct {
	Type          string `json:"type"`
	Listen        string `json:"listen"`
	ListenPort    int    `json:"listen_port"`
	Users         []User `json:"users"`
	TCPFastOpen   bool   `json:"tcp_fast_open"`
	UDPFragment   bool   `json:"udp_fragment"`
	Sniff         bool   `json:"sniff"`
	ProxyProtocol bool   `json:"proxy_protocol"`
}

type Config struct {
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
}

type Outbound struct {
	Type string `json:"type"`
}

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func generateVmessLink(inbound Inbound) string {
	user := inbound.Users[0] // Assuming there's only one user in the inbound
	// vmessLink := fmt.Sprintf("vmess://%s@%s:%d", user.UUID, inbound.Listen, inbound.ListenPort)

	// Generate base64 encoded JSON with necessary settings
	settingsJSON := fmt.Sprintf(`{
		"v": "2",
		"ps": "vmess",
		"add": "%s",
		"port": "%d",
		"id": "%s",
		"aid": %d,
		"net": "tcp",
		"type": "none",
		"host": "",
		"path": "",
		"tls": ""
	}`, inbound.Listen, inbound.ListenPort, user.UUID, user.AlterID)

	// Base64 encode the settings
	settingsBase64 := base64.StdEncoding.EncodeToString([]byte(settingsJSON))

	// Append settings to the vmess link
	vmessLink := fmt.Sprintf("vmess://%s", settingsBase64)

	return vmessLink
}

func generateAndWriteVmessLinks() {
	// 获取本机 IP 地址
	ip, err := getLocalIP()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		return
	}

	// 打开 JSON 文件
	file, err := os.Open("/usr/local/etc/sing-box/config.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 读取文件内容
	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// 解析 JSON
	var config Config
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// 创建文件用于写入 VMESS 链接
	outputFile, err := os.Create("/file/vmess_links.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer outputFile.Close()

	// 生成 VMESS 链接并写入文件
	for _, inbound := range config.Inbounds {
		if inbound.Type == "vmess" {
			inbound.Listen = ip
			vmessLink := generateVmessLink(inbound)
			_, err := outputFile.WriteString(vmessLink + "\n")
			if err != nil {
				fmt.Println("Error writing to file:", err)
				return
			}
		}
	}

}
func runDaily(task func()) {
	// 获取当前时间
	now := time.Now()

	// 计算下一个执行时间点
	next := now.Add(24 * time.Hour) // 每天运行一次

	// 计算距离下一个执行时间点的等待时间
	waitDuration := next.Sub(now)

	// 创建定时器
	timer := time.NewTimer(waitDuration)

	// 等待定时器触发
	<-timer.C

	// 执行任务
	task()
}

func main() {
	// 第一次运行时，运行一次函数
	generateAndWriteVmessLinks()

	runDaily(func() {
		generateAndWriteVmessLinks()
	})
}
