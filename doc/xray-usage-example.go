// xray-usage-example.go
// 这是一个示例文件，展示如何在项目中使用 xray-core

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"myproxy.com/p/internal/xray"
)

// 示例 1: 从文件加载 xray 配置
func example1_LoadFromFile() {
	instance, err := xray.NewXrayInstanceFromFile("xray_config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer instance.Stop()

	if err := instance.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("Xray-core 实例已启动")
	
	// 保持运行
	select {}
}

// 示例 2: 从 JSON 字节创建 xray 实例
func example2_LoadFromJSON() {
	// 创建一个简单的 xray 配置
	configJSON := []byte(`{
		"log": {
			"loglevel": "warning"
		},
		"inbounds": [{
			"port": 10808,
			"protocol": "socks",
			"settings": {
				"auth": "noauth"
			}
		}],
		"outbounds": [{
			"protocol": "freedom",
			"settings": {}
		}]
	}`)

	instance, err := xray.NewXrayInstanceFromJSON(configJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer instance.Stop()

	if err := instance.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("Xray-core 实例已启动")
	
	// 保持运行
	select {}
}

// 示例 3: 使用 xray-core 作为代理客户端
func example3_UseAsProxy() {
	// 配置 xray-core 连接到一个 SOCKS5 服务器
	configJSON := []byte(`{
		"log": {
			"loglevel": "warning"
		},
		"outbounds": [{
			"tag": "proxy",
			"protocol": "socks",
			"settings": {
				"servers": [{
					"address": "127.0.0.1",
					"port": 1080
				}]
			}
		}]
	}`)

	instance, err := xray.NewXrayInstanceFromJSON(configJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer instance.Stop()

	if err := instance.Start(); err != nil {
		log.Fatal(err)
	}

	// 通过 xray-core 连接到目标地址
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := instance.DialContext(ctx, "tcp", "www.example.com:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Println("成功通过 xray-core 连接到目标")
}

// 示例 4: 动态创建 VMess 出站配置
func example4_VMessOutbound() {
	// 创建 VMess 出站配置
	vmessConfig, err := xray.CreateVMessOutbound(
		"vmess-out",
		"server.example.com",
		443,
		"uuid-here",
		"auto",
		0,
	)
	if err != nil {
		log.Fatal(err)
	}

	// 构建完整的 xray 配置
	fullConfig := map[string]interface{}{
		"log": map[string]string{
			"loglevel": "warning",
		},
		"outbounds": []interface{}{
			json.RawMessage(vmessConfig),
		},
	}

	configJSON, _ := json.Marshal(fullConfig)
	
	instance, err := xray.NewXrayInstanceFromJSON(configJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer instance.Stop()

	if err := instance.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("VMess 代理已启动")
}

// 示例 5: 创建带认证的 SOCKS5 出站
func example5_SOCKS5WithAuth() {
	socksConfig, err := xray.CreateSimpleSOCKS5Outbound(
		"socks-out",
		"proxy.example.com",
		1080,
		"username",
		"password",
	)
	if err != nil {
		log.Fatal(err)
	}

	// 构建完整配置
	fullConfig := map[string]interface{}{
		"log": map[string]string{
			"loglevel": "warning",
		},
		"outbounds": []interface{}{
			map[string]interface{}{
				"tag":      "socks-out",
				"protocol": "socks",
				"settings": json.RawMessage(socksConfig),
			},
		},
	}

	configJSON, _ := json.Marshal(fullConfig)
	
	instance, err := xray.NewXrayInstanceFromJSON(configJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer instance.Stop()

	if err := instance.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("带认证的 SOCKS5 代理已启动")
}

// 示例 6: 集成到现有的 Forwarder 中
func example6_IntegrateWithForwarder() {
	// 这个示例展示如何修改 internal/proxy/forwarder.go
	
	/*
	// 在 Forwarder 结构体中添加：
	type Forwarder struct {
		SOCKS5Client   *socks5.SOCKS5Client
		XrayInstance   *xray.XrayInstance  // 新增
		UseXray        bool                 // 新增：是否使用 xray
		// ... 其他字段
	}

	// 在 handleTCPConnection 方法中：
	func (f *Forwarder) handleTCPConnection(localConn net.Conn) {
		var proxyConn net.Conn
		var err error

		if f.UseXray && f.XrayInstance != nil {
			// 使用 xray-core
			proxyConn, err = f.XrayInstance.Dial("tcp", f.RemoteAddr)
			if err != nil {
				f.log("ERROR", "proxy", "通过 xray-core 连接失败: %v", err)
				return
			}
		} else {
			// 使用现有的 SOCKS5 客户端
			proxyConn, err = f.SOCKS5Client.Dial("tcp", f.RemoteAddr)
			if err != nil {
				f.log("ERROR", "proxy", "通过 SOCKS5 代理连接失败: %v", err)
				return
			}
		}
		defer proxyConn.Close()

		// ... 后续的双向转发逻辑保持不变
	}
	*/
	fmt.Println("示例代码请参考注释")
}

func main() {
	fmt.Println("Xray-core 集成示例")
	fmt.Println("请根据需求选择合适的示例函数")
}

