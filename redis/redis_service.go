package redis

import (
	"context"
	"fmt"
	"main/utils"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	prefixTenantBySubdomain  = "tenant:subdomain:"
	defaultTTL               = 24 * time.Hour
	initialReconnectInterval = 5 * time.Second // Начальный интервал для переподключения
	maxReconnectInterval     = 5 * time.Minute // Максимальный интервал для переподключения
	reconnectMultiplier      = 2               // Множитель для экспоненциального backoff
)

// RedisUnavailableError represents an error when Redis is unavailable
type RedisUnavailableError struct {
	Err error
}

func (e *RedisUnavailableError) Error() string {
	return fmt.Sprintf("redis is unavailable: %v", e.Err)
}

// IsRedisUnavailable checks if the error is RedisUnavailableError
func IsRedisUnavailable(err error) bool {
	_, ok := err.(*RedisUnavailableError)
	return ok
}

// RedisConfig stores Redis configuration parameters
type RedisConfig struct {
	Host            string
	Port            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	MaxRetries      int
	MinRetryBackoff time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PoolTimeout     time.Duration
	IdleTimeout     time.Duration
	MaxConnAge      time.Duration
}

// NewRedisConfigFromEnv creates Redis configuration from environment variables
func NewRedisConfigFromEnv() *RedisConfig {
	return &RedisConfig{
		Host:            getEnvWithDefault("REDIS_HOST", "localhost"),
		Port:            getEnvWithDefault("REDIS_PORT", "6379"),
		Password:        os.Getenv("REDIS_PASSWORD"),
		DB:              getEnvInt("REDIS_DB", 0),
		PoolSize:        getEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns:    getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
		MaxRetries:      getEnvInt("REDIS_MAX_RETRIES", 3),
		MinRetryBackoff: getEnvDuration("REDIS_RETRY_BACKOFF", 100*time.Millisecond),
		DialTimeout:     getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:     getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout:    getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		PoolTimeout:     getEnvDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
		IdleTimeout:     getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		MaxConnAge:      getEnvDuration("REDIS_MAX_CONN_AGE", 0),
	}
}

type TenantCacheService struct {
	client       *redis.Client
	config       *RedisConfig
	mu           sync.RWMutex // Мьютекс для безопасного доступа к client
	healthCtx    context.Context
	healthCancel context.CancelFunc
	wg           sync.WaitGroup // WaitGroup для ожидания завершения горутин
}

var (
	instance *TenantCacheService
	once     sync.Once
)

// GetTenantCacheService returns a singleton instance of TenantCacheService
func GetTenantCacheService() (*TenantCacheService, error) {
	once.Do(func() {
		config := NewRedisConfigFromEnv()
		instance = &TenantCacheService{
			client: nil,
			config: config,
		}

		// Запуск горутины мониторинга здоровья соединения
		instance.healthCtx, instance.healthCancel = context.WithCancel(context.Background())

		// Добавляем горутину в WaitGroup перед запуском
		instance.wg.Add(1)
		go instance.healthCheckLoop()

		// Пытаемся установить начальное соединение
		if client, err := newRedisClient(config); err == nil {
			instance.setClient(client)
		}
	})

	// Проверяем текущее состояние соединения и возвращаем соответствующую ошибку
	if client := instance.getClient(); client != nil {
		return instance, nil
	}

	return instance, &RedisUnavailableError{Err: fmt.Errorf("redis client is nil")}
}

// healthCheckLoop периодически проверяет доступность Redis и восстанавливает соединение при необходимости
func (s *TenantCacheService) healthCheckLoop() {
	defer s.wg.Done()

	// Инициализируем генератор случайных чисел с уникальным seed
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	currentInterval := initialReconnectInterval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Проверяем текущее состояние
			if client := s.getClient(); client == nil {
				utils.Logger.Debug("Attempting to reconnect to Redis",
					zap.Duration("interval", currentInterval))

				if newClient, err := newRedisClient(s.config); err == nil {
					s.setClient(newClient)
					utils.Logger.Info("Successfully reconnected to Redis")

					// Сбрасываем интервал после успешного подключения
					currentInterval = initialReconnectInterval
					ticker.Reset(currentInterval)
				} else {
					utils.Logger.Debug("Failed to reconnect to Redis", zap.Error(err))

					// Увеличиваем интервал экспоненциально с добавлением джиттера
					currentInterval = time.Duration(float64(currentInterval) * reconnectMultiplier)
					if currentInterval > maxReconnectInterval {
						currentInterval = maxReconnectInterval
					}

					// Добавляем джиттер ±10% к интервалу
					jitter := time.Duration(rnd.Int63n(int64(currentInterval/5))) - currentInterval/10
					nextInterval := currentInterval + jitter

					utils.Logger.Debug("Next reconnect with jitter",
						zap.Duration("base_interval", currentInterval),
						zap.Duration("jitter", jitter),
						zap.Duration("next_interval", nextInterval))

					ticker.Reset(nextInterval)
				}
			} else {
				// Проверяем работоспособность существующего соединения
				// Используем производный контекст от healthCtx с таймаутом
				ctx, cancel := context.WithTimeout(s.healthCtx, 2*time.Second)
				if err := client.Ping(ctx).Err(); err != nil {
					utils.Logger.Warn("Redis connection is unhealthy, closing and will attempt to reconnect",
						zap.Error(err))
					client.Close()
					s.setClient(nil)

					// Устанавливаем начальный интервал для новой попытки
					currentInterval = initialReconnectInterval
					ticker.Reset(currentInterval)
				}
				cancel()
			}
		case <-s.healthCtx.Done():
			utils.Logger.Debug("Redis health check loop stopped")
			return
		}
	}
}

// setClient безопасно устанавливает клиента Redis
func (s *TenantCacheService) setClient(client *redis.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
}

// getClient безопасно получает клиента Redis
func (s *TenantCacheService) getClient() *redis.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// Добавляю публичный метод для получения клиента Redis
func (s *TenantCacheService) GetClient() *redis.Client {
	return s.getClient()
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// newRedisClient creates new Redis client instance
func newRedisClient(config *RedisConfig) (*redis.Client, error) {
	utils.Logger.Debug("Initializing Redis connection",
		zap.String("host", config.Host),
		zap.String("port", config.Port),
		zap.String("password_set", map[bool]string{true: "yes", false: "no"}[config.Password != ""]),
	)

	opts := &redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.Host, config.Port),
		DB:   config.DB,
	}

	// Добавляем пароль только если он указан
	if config.Password != "" {
		opts.Password = config.Password
	}

	opts.PoolSize = config.PoolSize
	opts.MinIdleConns = config.MinIdleConns
	opts.MaxRetries = config.MaxRetries
	opts.MinRetryBackoff = config.MinRetryBackoff
	opts.DialTimeout = config.DialTimeout
	opts.ReadTimeout = config.ReadTimeout
	opts.WriteTimeout = config.WriteTimeout
	opts.PoolTimeout = config.PoolTimeout
	opts.IdleTimeout = config.IdleTimeout
	opts.MaxConnAge = config.MaxConnAge

	utils.Logger.Debug("Redis connection options",
		zap.Int("db", opts.DB),
		zap.Int("pool_size", opts.PoolSize),
		zap.Int("max_retries", opts.MaxRetries),
		zap.Duration("dial_timeout", opts.DialTimeout),
	)

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		utils.Logger.Warn("Redis is not available",
			zap.Error(err),
			zap.String("host", config.Host),
			zap.String("port", config.Port),
		)
		return nil, fmt.Errorf("failed to connect to Redis at %s:%s: %w", config.Host, config.Port, err)
	}

	utils.Logger.Info("Successfully connected to Redis",
		zap.String("host", config.Host),
		zap.String("port", config.Port),
		zap.Int("db", opts.DB),
		zap.Int("pool_size", opts.PoolSize),
	)

	return client, nil
}

// GetTenantSubdomainKey returns Redis key for tenant by subdomain
func GetTenantSubdomainKey(subdomain string) string {
	return prefixTenantBySubdomain + subdomain
}

// SetTenantCache stores tenant data in Redis cache
func (s *TenantCacheService) SetTenantCache(ctx context.Context, tenantID, cacheKey string, data []byte) error {
	client := s.getClient()
	if client == nil {
		return &RedisUnavailableError{Err: fmt.Errorf("redis client is nil")}
	}

	key := cacheKey
	if err := client.Set(ctx, key, data, defaultTTL).Err(); err != nil {
		utils.Logger.Warn("Failed to set tenant data in Redis",
			zap.Error(err),
			zap.String("tenant_id", tenantID),
			zap.String("cache_key", cacheKey),
		)
		return &RedisUnavailableError{Err: err}
	}

	utils.Logger.Debug("Successfully cached tenant data in Redis",
		zap.String("tenant_id", tenantID),
		zap.String("cache_key", cacheKey),
		zap.Duration("ttl", defaultTTL),
	)

	return nil
}

// GetTenantCache retrieves tenant data from Redis cache
func (s *TenantCacheService) GetTenantCache(ctx context.Context, cacheKey string) ([]byte, error) {
	client := s.getClient()
	if client == nil {
		return nil, &RedisUnavailableError{Err: fmt.Errorf("redis client is nil")}
	}

	data, err := client.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("cache miss")
		}
		return nil, &RedisUnavailableError{Err: err}
	}

	return data, nil
}

// RefreshTenantCache refreshes the TTL of tenant data in Redis cache
func (s *TenantCacheService) RefreshTenantCache(ctx context.Context, tenantID, cacheKey string) error {
	client := s.getClient()
	if client == nil {
		return &RedisUnavailableError{Err: fmt.Errorf("redis client is nil")}
	}

	// Используем одну атомарную команду вместо Exists + Expire
	success, err := client.Expire(ctx, cacheKey, defaultTTL).Result()
	if err != nil {
		return &RedisUnavailableError{Err: err}
	}

	if !success {
		return fmt.Errorf("cache key does not exist")
	}

	utils.Logger.Debug("Successfully refreshed tenant data TTL in Redis",
		zap.String("tenant_id", tenantID),
		zap.String("cache_key", cacheKey),
		zap.Duration("ttl", defaultTTL),
	)

	return nil
}

// Close closes Redis connection and stops the health check
func (s *TenantCacheService) Close() error {
	// Останавливаем health check loop
	if s.healthCancel != nil {
		s.healthCancel()
	}

	// Дожидаемся завершения горутины мониторинга
	s.wg.Wait()

	// Закрываем клиент Redis
	client := s.getClient()
	if client == nil {
		return nil
	}
	return client.Close()
}
