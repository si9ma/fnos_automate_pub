package main

import (
	"fnos_photo_proxy/internal/config"
	"fnos_photo_proxy/internal/proxy"
	"log"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 创建代理服务器
	server := proxy.NewProxyServer(cfg)

	// 启动服务器
	log.Printf("Starting proxy server on port %s", cfg.Port)
	if err := server.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
