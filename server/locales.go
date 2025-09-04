package server

import (
	"encoding/json"
	"main/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

// InitI18n инициализирует систему интернационализации
func InitI18n() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	if err := LoadTranslations(bundle); err != nil {
		utils.Logger.Error("Failed to load translations", zap.Error(err))
		return nil, err
	}

	utils.Logger.Info("Translations loaded successfully")
	return bundle, nil
}

// LoadTranslations загружает все JSON файлы локализации
func LoadTranslations(bundle *i18n.Bundle) error {
	// Определяем правильный путь к директории локализации
	localesDir := findLocalesDir()

	// Проверяем существование директории локализаций
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		utils.Logger.Warn("Locales build directory not found", zap.String("path", localesDir))
		return nil
	}

	utils.Logger.Info("Loading translations from directory", zap.String("path", localesDir))

	return filepath.Walk(localesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			utils.Logger.Debug("Loading translation file", zap.String("file", path))
			jsonFile, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = bundle.ParseMessageFileBytes(jsonFile, path)
			return err
		}
		return nil
	})
}

// findLocalesDir находит правильную директорию локализации
func findLocalesDir() string {
	// Пробуем разные пути
	paths := []string{
		"locales/build",       // Обычное использование
		"../../locales/build", // Для тестов из tests/integration
		"../locales/build",    // Для тестов из tests/
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Если ничего не найдено, возвращаем стандартный путь
	return "locales/build"
}
