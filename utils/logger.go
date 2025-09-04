package utils

import (
	// "main/querylog" // TODO: uncomment after creation
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogger init logger
func InitLogger() {
	config := zap.NewProductionConfig()

	// Set output path
	config.OutputPaths = []string{"stdout"}

	// Set time format
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Set log level depending on environment
	if os.Getenv("GO_ENV") == "production" {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		// Additional settings for production
		config.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
	} else {
		// For local development
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
		config.Encoding = "console" // More readable format for development

		// Set color output for console
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Создаем базовый логгер с оригинальными опциями
	options := []zap.Option{
		zap.AddCallerSkip(0), // Изменено с 1 на 0 для правильного caller
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	// Если включено логирование запросов, добавляем обертку
	if os.Getenv("ENABLE_QUERY_LOG") == "true" && os.Getenv("ENV") != "production" {
		options = append(options, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			// TODO: uncomment after querylog creation
			// return querylog.NewQueryLogCore(core, querylog.GetCollector())
			return core
		}))
	}

	var err error
	Logger, err = config.Build(options...)
	if err != nil {
		panic(err)
	}
}

// DebugLog логирует debug-сообщения с поддержкой форматирования
func DebugLog(ctx interface{}, format string, args ...interface{}) {
	if Logger != nil {
		Logger.Sugar().Debugf(format, args...)
	}
}
