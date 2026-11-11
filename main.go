package main

import (
	"collect-modbus/config"
	"collect-modbus/internal/service"
	"collect-modbus/pkgs/env"
	"collect-modbus/pkgs/logger"
	"collect-modbus/pkgs/shutdown"

	"go.uber.org/zap"
)

func main() {
	// 未通过 ldflags 注入时使用项目默认值
	if env.AppName == "" {
		env.AppName = "collect-modbus"
	}

	if _, err := config.Load("config/config.toml"); err != nil {
		panic(err)
	}

	log, err := logger.NewLogger("root",
		logger.WithLevel(config.Get().Log.Level),
		logger.WithFileRotationP(config.Get().Log.Path, config.Get().Log.MaxSize,
			config.Get().Log.MaxBackups, config.Get().Log.MaxAge),
		logger.WithDisableConsole(!config.Get().Log.Console),
	)
	if err != nil {
		panic(err)
	}

	info := env.GetInfo()
	log.Info("collect-modbus start",
		zap.String("app", info.AppName),
		zap.String("version", info.Version),
		zap.String("gitCommit", info.GitCommit),
		zap.String("buildDate", info.BuildDate),
		zap.String("goVersion", info.GoVersion),
		zap.String("compiler", info.Compiler),
		zap.String("platform", info.Platform),
	)
	service := service.NewService(log)
	if err := service.Start(); err != nil {
		panic(err)
	}

	// 注册优雅退出，阻塞等待 SIGINT/SIGTERM
	shutdown.NewHook().Close(
		func() {
			log.Info("collect-modbus stopped")
			logger.CloseAll()
		},
	)
}
