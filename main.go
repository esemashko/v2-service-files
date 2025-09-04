package main

import (
	"context"
	"flag"
	_ "main/ent/runtime"
	"main/middleware"
	"main/server"
	"main/utils"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fmt"
	"net/http"

	"main/redis"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	exportSchema := flag.Bool("schema", false, "Export GraphQL schema to schema.graphql")
	flag.Parse()

	// Load environment variables BEFORE initializing logger
	if err := godotenv.Load(".env"); err != nil {
		// Use fmt for initial logging since logger is not initialized yet
		fmt.Printf("No .env file found, using environment variables: %v\n", err)
	}

	// Initialize logger AFTER loading environment variables
	utils.InitLogger()
	defer utils.Logger.Sync()

	// Настраиваем graceful shutdown
	// Перехватываем сигналы завершения программы (Ctrl+C, kill, и т.д.)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Export GraphQL schema
	if *exportSchema {
		if err := server.ExportSchema(); err != nil {
			utils.Logger.Fatal("Error exporting schema",
				zap.Error(err),
			)
		}
		return
	}

	// Run web server with graceful shutdown
	runWebServerWithGracefulShutdown(shutdown)
}

func runWebServerWithGracefulShutdown(shutdown chan os.Signal) {
	// Setup router with GraphQL server
	router, err := server.SetupRouter()
	if err != nil {
		utils.Logger.Fatal("Failed to setup router",
			zap.Error(err))
	}

	port := os.Getenv("APP_CORE_PORT")
	if port == "" {
		port = "9010" // Default port if not specified
	}

	// Создаем HTTP-сервер
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		utils.Logger.Info(fmt.Sprintf("Server started on port %s", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Logger.Fatal("Server startup failed",
				zap.Error(err),
			)
		}
	}()

	// Ожидаем сигнал завершения
	<-shutdown
	utils.Logger.Info("Shutdown signal received, gracefully shutting down...")

	// Создаем единый контекст с таймаутом для всего процесса shutdоwn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Подготавливаем блок для сброса логов
	flushLogs := func() {
		if err := utils.Logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Error flushing logs: %v\n", err)
		}
	}

	// 1. Сначала останавливаем HTTP-сервер
	serverCtx, serverCancel := context.WithTimeout(ctx, 15*time.Second)
	defer serverCancel()

	if err := srv.Shutdown(serverCtx); err != nil {
		utils.Logger.Error("Server shutdown error",
			zap.Error(err),
		)
	} else {
		utils.Logger.Info("Server shutdown complete")
	}

	// Сбрасываем логи после остановки сервера
	flushLogs()

	// 2. Закрываем соединения с БД
	if err := middleware.CloseDatabaseClient(); err != nil {
		utils.Logger.Error("Database shutdown error",
			zap.Error(err),
		)
	} else {
		utils.Logger.Info("Database shutdown complete")
	}

	// 3. Закрываем Redis-соединение
	if cacheService, err := redis.GetTenantCacheService(); err == nil {
		if err := cacheService.Close(); err != nil {
			utils.Logger.Error("Redis shutdown error",
				zap.Error(err),
			)
		} else {
			utils.Logger.Info("Redis shutdown complete")
		}
	}

	// Финальный сброс логов
	flushLogs()

	utils.Logger.Info("Graceful shutdown complete")
	flushLogs()
}
