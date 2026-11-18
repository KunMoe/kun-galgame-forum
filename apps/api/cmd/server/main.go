package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"kun-galgame-api/internal/agentmcp"
	"kun-galgame-api/internal/app"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/health"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	port, _ := strconv.Atoi(config.Env("SERVER_PORT", "2334"))
	health.MaybeProbe(port, "/healthz")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	logger.Init(cfg.Server.Mode)

	application := app.New(cfg)

	agentMCP := &http.Server{Addr: cfg.Agent.MCPAddr, Handler: application.AgentMCP, ReadHeaderTimeout: 10 * time.Second}
	if agentMCP.Addr != "" {
		go func() {
			slog.Info("AI 助手站点工具 (MCP) 启动", "addr", agentMCP.Addr, "path", agentmcp.Path)
			if err := agentMCP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("AI 助手站点工具 (MCP) 启动失败", "error", err)
			}
		}()
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("正在关闭服务器...")
		_ = agentMCP.Close()
		_ = application.Fiber.Shutdown()
	}()

	addr := ":" + cfg.Server.Port
	slog.Info("服务器启动", "addr", addr)
	if err := application.Fiber.Listen(addr); err != nil {
		slog.Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}
