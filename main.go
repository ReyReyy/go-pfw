package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// 版本信息
const (
	Version = "v0.1.0-beta.4"
)

// 配置结构
type NetworkConfig struct {
	Type        interface{} `yaml:"type" json:"type"`
	SendProxy   *bool       `yaml:"send_proxy" json:"send_proxy"`
	AcceptProxy *bool       `yaml:"accept_proxy" json:"accept_proxy"`
}

type GlobalConfig struct {
	LogLevel string        `yaml:"loglevel" json:"loglevel"`
	Network  NetworkConfig `yaml:"network" json:"network"`
}

type Service struct {
	Name    string        `yaml:"name" json:"name"`
	Listen  string        `yaml:"listen" json:"listen"`
	Remote  string        `yaml:"remote" json:"remote"`
	Network NetworkConfig `yaml:"network" json:"network"`
}

type Config struct {
	Global   GlobalConfig `yaml:"global" json:"global"`
	Services []Service    `yaml:"services" json:"services"`
}

// 全局变量
var (
	logLevel    = "info"
	debugMode   = false
	showHelp    = false
	showVersion = false
)

func main() {
	// 命令行参数解析
	var (
		configPath  string
		listenAddr  string
		remoteAddr  string
		networkType string
		sendProxy   bool
		acceptProxy bool
	)

	flag.StringVar(&configPath, "c", "", "Path to config file")
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&listenAddr, "l", "", "Listen address")
	flag.StringVar(&listenAddr, "listen", "", "Listen address")
	flag.StringVar(&remoteAddr, "r", "", "Remote address")
	flag.StringVar(&remoteAddr, "remote", "", "Remote address")
	flag.StringVar(&networkType, "n", "tcp", "Network type (tcp/udp/both)")
	flag.StringVar(&networkType, "type", "tcp", "Network type (tcp/udp/both)")
	flag.BoolVar(&sendProxy, "send_proxy", false, "Enable sending PROXY protocol")
	flag.BoolVar(&acceptProxy, "accept_proxy", false, "Accept PROXY protocol")
	flag.BoolVar(&debugMode, "d", false, "Enable debug mode")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.Usage = printHelp

	// 自定义参数解析
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if strings.Contains(err.Error(), "flag provided but not defined") {
			log.Fatalf("Unknown flag: %v", err)
		}
		printHelp()
		os.Exit(1)
	}

	if showVersion {
		fmt.Println("go-pfw version:", Version)
		return
	}

	if showHelp {
		printHelp()
		return
	}

	// 初始化日志
	if debugMode {
		logLevel = "debug"
	}

	// 加载配置
	var services []Service
	if configPath != "" {
		cfg := loadConfigFile(configPath)
		logLevel = getLogLevel(cfg.Global.LogLevel)
		services = parseServices(cfg)
	} else {
		if listenAddr == "" || remoteAddr == "" {
			log.Fatal("Missing required parameters: --listen and --remote")
		}

		networkConfig := NetworkConfig{
			Type:        networkType,
			SendProxy:   &sendProxy,
			AcceptProxy: &acceptProxy,
		}

		services = []Service{{
			Name:    "",
			Listen:  listenAddr,
			Remote:  remoteAddr,
			Network: networkConfig,
		}}
	}

	// 启动服务
	for _, svc := range services {
		startService(svc)
	}

	select {} // 保持主线程运行
}

// 辅助函数
func printHelp() {
	fmt.Printf(`Port Forward Tool %s - Usage:

Options:
  -c, --config <path>     配置文件路径
  -l, --listen <addr>     监听地址 (无配置文件时必需)
  -r, --remote <addr>     目标地址 (无配置文件时必需)
  -n, --type <type>       网络类型 [tcp|udp|both] (默认tcp)
      --send_proxy        启用发送PROXY协议
      --accept_proxy      接受PROXY协议
  -d, --debug             调试模式
  -h, --help              显示帮助
  -v, --version           显示版本

`, Version)
	os.Exit(0)
}

func loadConfigFile(path string) Config {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error opening config: %v", err)
	}
	defer file.Close()

	var cfg Config
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
			log.Fatalf("Error decoding YAML: %v", err)
		}
	case ".json":
		if err := json.NewDecoder(file).Decode(&cfg); err != nil {
			log.Fatalf("Error decoding JSON: %v", err)
		}
	default:
		log.Fatalf("Unsupported config format: %s", ext)
	}
	return cfg
}

func parseServices(cfg Config) []Service {
	var services []Service
	for _, svc := range cfg.Services {
		// 合并全局配置
		if svc.Network.Type == nil {
			svc.Network.Type = cfg.Global.Network.Type
		}

		// 当服务级别的SendProxy未设置时，继承全局配置
		if svc.Network.SendProxy == nil {
			svc.Network.SendProxy = cfg.Global.Network.SendProxy
		}

		// 当服务级别的AcceptProxy未设置时，继承全局配置
		if svc.Network.AcceptProxy == nil {
			svc.Network.AcceptProxy = cfg.Global.Network.AcceptProxy
		}

		services = append(services, svc)
		logDebug(svc.Name, "Service loaded: %+v", svc)
	}
	return services
}

func startService(svc Service) {
	listen, err := parseAddress(svc.Listen, true)
	if err != nil {
		logError(svc.Name, "Invalid listen address: %v", err)
		return
	}

	remote, err := parseAddress(svc.Remote, false)
	if err != nil {
		logError(svc.Name, "Invalid remote address: %v", err)
		return
	}

	networkTypes, err := parseNetworkType(svc.Network.Type)
	if err != nil {
		logError(svc.Name, "Invalid network: %v", err)
		return
	}

	for _, netType := range networkTypes {
		if netType == "udp" && (svc.Network.SendProxy != nil && *svc.Network.SendProxy || svc.Network.AcceptProxy != nil && *svc.Network.AcceptProxy) {
			logError(svc.Name, "Proxy Protocol not supported for UDP")
			continue
		}

		switch netType {
		case "tcp":
			go startTCPForwarder(svc, listen, remote)
		case "udp":
			go startUDPForwarder(svc, listen, remote)
		}
	}
}

// 网络处理函数
func startTCPForwarder(svc Service, listen, remote string) {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		logError(svc.Name, "TCP listen error: %v", err)
		return
	}
	defer ln.Close() // 确保监听器关闭

	var flags []string
	if svc.Network.SendProxy != nil && *svc.Network.SendProxy {
		flags = append(flags, "send_proxy:true")
	}
	if svc.Network.AcceptProxy != nil && *svc.Network.AcceptProxy {
		flags = append(flags, "accept_proxy:true")
	}

	logMsg := fmt.Sprintf("TCP %s -> %s", listen, remote)
	if len(flags) > 0 {
		logMsg += fmt.Sprintf(" (%s)", strings.Join(flags, " "))
	}
	logInfo(svc.Name, logMsg)

	for {
		conn, err := ln.Accept()
		if err != nil {
			logError(svc.Name, "Accept error: %v", err)
			continue
		}
		logDebug(svc.Name, "New TCP connection from %s", conn.RemoteAddr())
		go handleTCPConnection(svc, conn, remote)
	}
}

func handleTCPConnection(svc Service, src net.Conn, remote string) {
	defer src.Close()

	var proxyInfo *ProxyInfo
	if svc.Network.AcceptProxy != nil && *svc.Network.AcceptProxy {
		reader := bufio.NewReader(src)
		header, err := reader.ReadString('\n')
		if err != nil {
			logError(svc.Name, "Proxy header read error: %v", err)
			return
		}

		proxyInfo, err = parseProxyHeader(header)
		if err != nil {
			logError(svc.Name, "Invalid proxy header: %v", err)
			return
		}
	}

	dst, err := net.Dial("tcp", remote)
	if err != nil {
		logError(svc.Name, "Remote connect error: %v", err)
		return
	}
	defer dst.Close()

	// 设置读写超时
	src.SetDeadline(time.Now().Add(5 * time.Minute))
	dst.SetDeadline(time.Now().Add(5 * time.Minute))

	if svc.Network.SendProxy != nil && *svc.Network.SendProxy {
		var srcAddr, dstAddr net.Addr
		if proxyInfo != nil {
			srcAddr = &net.TCPAddr{
				IP:   net.ParseIP(proxyInfo.SrcIP),
				Port: proxyInfo.SrcPort,
			}
			dstAddr = &net.TCPAddr{
				IP:   net.ParseIP(proxyInfo.DstIP),
				Port: proxyInfo.DstPort,
			}
		} else {
			srcAddr = src.RemoteAddr()
			dstAddr = src.LocalAddr()
		}

		header := buildProxyHeader(srcAddr, dstAddr)
		if _, err := dst.Write(header); err != nil {
			logError(svc.Name, "Proxy header write error: %v", err)
			return
		}
	}

	var srcReader io.Reader = src
	if svc.Network.AcceptProxy != nil && *svc.Network.AcceptProxy {
		srcReader = io.MultiReader(bytes.NewReader([]byte{}), src)
	}

	go func() {
		defer dst.Close()
		io.Copy(dst, srcReader)
		src.Close()
	}()
	io.Copy(src, dst)
	dst.Close()
}

// UDP转发实现
func startUDPForwarder(svc Service, listen, remote string) {
	lnAddr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		logError(svc.Name, "UDP resolve error: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", lnAddr)
	if err != nil {
		logError(svc.Name, "UDP listen error: %v", err)
		return
	}
	defer conn.Close()

	remoteAddr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		logError(svc.Name, "Remote UDP resolve error: %v", err)
		return
	}

	logInfo(svc.Name, "UDP %s -> %s", listen, remote)

	// 使用缓冲通道控制并发量
	sem := make(chan struct{}, 1000) // 最大并发1000

	for {
		buf := make([]byte, 4096)
		n, cliAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			logError(svc.Name, "UDP read error: %v", err)
			continue
		}
		logDebug(svc.Name, "Received %d bytes from %s", n, cliAddr)

		sem <- struct{}{}
		go func(data []byte, clientAddr *net.UDPAddr) {
			defer func() { <-sem }()

			// 直接使用原始连接回复
			resp, err := forwardUDP(data, remoteAddr)
			if err != nil {
				logError(svc.Name, "UDP forward error: %v", err)
				return
			}

			if _, err := conn.WriteToUDP(resp, clientAddr); err != nil {
				logError(svc.Name, "UDP reply error: %v", err)
			} else {
				logDebug(svc.Name, "Replied %d bytes to %s", len(resp), clientAddr)
			}
		}(buf[:n], cliAddr)
	}
}

func forwardUDP(data []byte, remoteAddr *net.UDPAddr) ([]byte, error) {
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.Write(data); err != nil {
		return nil, err
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	resp := make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil {
		return nil, err
	}

	return resp[:n], nil
}

// 日志工具
func getLogLevel(level string) string {
	if debugMode {
		return "debug"
	}
	if level == "" {
		return "info"
	}
	return strings.ToLower(level)
}

func logInfo(serviceName, format string, v ...interface{}) {
	if logLevel == "none" {
		return
	}
	if serviceName != "" {
		format = "[" + serviceName + "] " + format
	}
	log.Printf("[INFO] "+format, v...)
}

func logError(serviceName, format string, v ...interface{}) {
	if serviceName != "" {
		format = "[" + serviceName + "] " + format
	}
	log.Printf("[ERROR] "+format, v...)
}

func logDebug(serviceName, format string, v ...interface{}) {
	if logLevel == "debug" {
		if serviceName != "" {
			format = "[" + serviceName + "] " + format
		}
		log.Printf("[DEBUG] "+format, v...)
	}
}

// 网络地址解析
func parseAddress(addr string, isListen bool) (string, error) {
	if isListen && !strings.Contains(addr, ":") {
		iface, err := net.InterfaceByName(addr)
		if err != nil {
			return "", fmt.Errorf("invalid interface: %w", err)
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", fmt.Errorf("interface addresses: %w", err)
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return ipnet.IP.String() + ":0", nil
			}
		}
		return "", fmt.Errorf("no IPv4 address found for interface")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	if ip := net.ParseIP(host); ip == nil {
		if ips, err := net.LookupIP(host); err == nil {
			for _, ip := range ips {
				if ip.To4() != nil {
					return net.JoinHostPort(ip.String(), port), nil
				}
			}
		}
	}
	return addr, nil
}

// 网络类型解析
func parseNetworkType(networkType interface{}) ([]string, error) {
	// 处理空值情况
	if networkType == nil {
		return []string{"tcp"}, nil
	}

	// 根据输入类型处理
	switch v := networkType.(type) {
	case string:
		// 空字符串情况
		if v == "" {
			return []string{"tcp"}, nil
		}

		// 处理字符串类型
		v = strings.ToLower(v)
		switch v {
		case "both":
			return []string{"tcp", "udp"}, nil
		case "tcp", "udp":
			return []string{v}, nil
		default:
			// 处理 "[tcp,udp]" 格式的字符串
			if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				networks := strings.Split(strings.Trim(v, "[]"), ",")
				var result []string
				for _, n := range networks {
					n = strings.TrimSpace(n)
					if n != "tcp" && n != "udp" {
						return nil, fmt.Errorf("invalid network type in array: %s", n)
					}
					result = append(result, n)
				}
				return result, nil
			}
			return nil, fmt.Errorf("invalid network type: %s", v)
		}

	case []string:
		// 空数组情况
		if len(v) == 0 {
			return []string{"tcp"}, nil
		}

		// 处理字符串数组
		var result []string
		for _, n := range v {
			n = strings.ToLower(strings.TrimSpace(n))
			if n != "tcp" && n != "udp" {
				return nil, fmt.Errorf("invalid network type in array: %s", n)
			}
			result = append(result, n)
		}
		return result, nil

	case []interface{}:
		// 处理通用数组类型（可能来自JSON解析等）
		if len(v) == 0 {
			return []string{"tcp"}, nil
		}

		var result []string
		for _, item := range v {
			// 尝试将元素转换为字符串
			if str, ok := item.(string); ok {
				str = strings.ToLower(strings.TrimSpace(str))
				if str != "tcp" && str != "udp" {
					return nil, fmt.Errorf("invalid network type in array: %s", str)
				}
				result = append(result, str)
			} else {
				return nil, fmt.Errorf("array contains non-string element")
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported network type: %T", networkType)
	}
}

// Proxy协议相关
type ProxyInfo struct {
	SrcIP   string
	SrcPort int
	DstIP   string
	DstPort int
}

func parseProxyHeader(header string) (*ProxyInfo, error) {
	parts := strings.Split(strings.TrimSpace(header), " ")
	if len(parts) < 6 || parts[0] != "PROXY" {
		return nil, fmt.Errorf("invalid proxy header format")
	}

	srcPort, _ := strconv.Atoi(parts[4])
	dstPort, _ := strconv.Atoi(parts[5])
	return &ProxyInfo{
		SrcIP:   parts[2],
		SrcPort: srcPort,
		DstIP:   parts[3],
		DstPort: dstPort,
	}, nil
}

func buildProxyHeader(src, dst net.Addr) []byte {
	srcTCP := src.(*net.TCPAddr)
	dstTCP := dst.(*net.TCPAddr)

	proto := "TCP4"
	if srcTCP.IP.To4() == nil {
		proto = "TCP6"
	}

	return []byte(fmt.Sprintf(
		"PROXY %s %s %s %d %d\r\n",
		proto,
		srcTCP.IP,
		dstTCP.IP,
		srcTCP.Port,
		dstTCP.Port,
	))
}
