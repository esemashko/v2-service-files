package utils

import (
	"context"
	"sync"

	federation "github.com/esemashko/v2-federation"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

var (
	i18nBundle *i18n.Bundle
	// Кеш локализаторов для разных языков (глобальный, безопасный для мультитенантности)
	// Кешируются только инструменты перевода, не данные пользователей
	localizerCache = make(map[string]*i18n.Localizer)
	localizerMutex sync.RWMutex
)

// SetI18nBundle устанавливает глобальный bundle для локализации
func SetI18nBundle(bundle *i18n.Bundle) {
	i18nBundle = bundle
	// Очищаем кеш при установке нового bundle
	localizerMutex.Lock()
	localizerCache = make(map[string]*i18n.Localizer)
	localizerMutex.Unlock()
}

// GetI18nBundle возвращает глобальный bundle для локализации
func GetI18nBundle() *i18n.Bundle {
	return i18nBundle
}

// getLocalizer возвращает закешированный локализатор или создает новый
func getLocalizer(lang string) *i18n.Localizer {
	// Быстрая проверка с read lock
	localizerMutex.RLock()
	if localizer, ok := localizerCache[lang]; ok {
		localizerMutex.RUnlock()
		return localizer
	}
	localizerMutex.RUnlock()

	// Создаем новый локализатор с write lock
	localizerMutex.Lock()
	defer localizerMutex.Unlock()

	// Проверяем еще раз после получения write lock (double-check pattern)
	if localizer, ok := localizerCache[lang]; ok {
		return localizer
	}

	// Парсим язык тег
	langTag, err := language.Parse(lang)
	if err != nil {
		langTag = language.English
	}

	// Создаем и кешируем локализатор
	localizer := i18n.NewLocalizer(GetI18nBundle(), langTag.String())
	localizerCache[lang] = localizer

	return localizer
}

// TemplateData представляет данные для подстановки в шаблон локализации
type TemplateData map[string]interface{}

// T возвращает локализованную строку по ключу с подстановкой переменных
func T(ctx context.Context, messageID string, data ...TemplateData) string {
	// Получаем язык из federation контекста
	lang := federation.GetLanguage(ctx)
	if lang == "" {
		lang = "en"
	}

	// Получаем закешированный локализатор
	localizer := getLocalizer(lang)
	if localizer == nil {
		Logger.Error("Failed to get localizer",
			zap.String("messageID", messageID),
			zap.String("language", lang),
		)
		return messageID
	}

	config := &i18n.LocalizeConfig{
		MessageID: messageID,
	}

	if len(data) > 0 {
		config.TemplateData = data[0]
	}

	msg, err := localizer.Localize(config)
	if err != nil {
		Logger.Error("Failed to localize message",
			zap.String("messageID", messageID),
			zap.Error(err),
		)
		return messageID
	}

	return msg
}
