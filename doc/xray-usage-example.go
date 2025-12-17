// xray-usage-example.go
// 这是一个示例文件，展示如何在项目中使用 xray-core

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"myproxy.com/p/internal/xray"
)

// 示例 1: 从文件加载 xray 配置
// 注意：NewXrayInstanceFromFile 已移除，请使用 NewXrayInstanceFromJSON
func example1_LoadFromFile() {
	// 读取配置文件
	configBytes, err := os.ReadFile("xray_config.json")
	if err != nil {
		log.Fatal(err)
	}
	
	instance, err := xray.NewXrayInstanceFromJSON(configBytes)
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

	// 注意：DialContext 和 Dial 方法已移除
	// xray-core 通过配置自动处理所有连接，不需要手动调用连接方法
	log.Println("Xray-core 实例已启动，连接将通过配置自动处理")
}

// 示例 4: 动态创建 VMess 出站配置
// 注意：CreateVMessOutbound 已移除，请使用 CreateOutboundFromServer 或手动构建配置
func example4_VMessOutbound() {
	// 手动构建 VMess 出站配置
	vmessConfig := map[string]interface{}{
		"tag":      "vmess-out",
		"protocol": "vmess",
		"settings": map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": "server.example.com",
					"port":    443,
					"users": []map[string]interface{}{
						{
							"id":       "uuid-here",
							"alterId":  0,
							"security": "auto",
						},
					},
				},
			},
		},
	}

	// 构建完整的 xray 配置
	fullConfig := map[string]interface{}{
		"log": map[string]string{
			"loglevel": "warning",
		},
		"outbounds": []interface{}{
			vmessConfig,
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
// 注意：CreateSimpleSOCKS5Outbound 已移除，请手动构建配置
func example5_SOCKS5WithAuth() {
	// 手动构建 SOCKS5 出站配置
	socksConfig := map[string]interface{}{
		"tag":      "socks-out",
		"protocol": "socks",
		"settings": map[string]interface{}{
			"auth": "password",
			"accounts": []map[string]string{
				{
					"user": "username",
					"pass": "password",
				},
			},
			"servers": []map[string]interface{}{
				{
					"address": "proxy.example.com",
					"port":    1080,
				},
			},
		},
	}

	// 构建完整配置
	fullConfig := map[string]interface{}{
		"log": map[string]string{
			"loglevel": "warning",
		},
		"outbounds": []interface{}{
			socksConfig,
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
			// 注意：Dial 方法已移除，xray-core 通过配置自动处理连接
			// 请使用配置方式启动 xray-core，它会自动处理所有连接
			// proxyConn, err = f.XrayInstance.Dial("tcp", f.RemoteAddr)
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

