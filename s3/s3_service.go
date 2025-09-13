package s3

import (
	"context"
	"fmt"
	"io"
	"main/ctxkeys"
	"main/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// S3Service handles S3 operations for tenant files
type S3Service struct {
}

// S3Config contains S3 configuration
type S3Config struct {
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Endpoint  string
	UseSSL    bool
	PathStyle string
	UserName  string
}

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// NewS3Service creates a new S3 service instance
func NewS3Service() *S3Service {
	return &S3Service{}
}

// getS3Client creates an S3 client with given configuration
func (s *S3Service) getS3Client(config *S3Config) (*s3.S3, error) {
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("S3 credentials are not configured")
	}

	awsConfig := &aws.Config{
		Region:      aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
	}

	// Set endpoint for MinIO or custom S3-compatible storage
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
		awsConfig.DisableSSL = aws.Bool(!config.UseSSL)

		// Force path style for MinIO
		if config.PathStyle == "path" || config.PathStyle == "auto" {
			awsConfig.S3ForcePathStyle = aws.Bool(true)
		}
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return s3.New(sess), nil
}

// getTenantS3Config returns S3 configuration for the tenant from context
func (s *S3Service) getTenantS3Config(ctx context.Context) (*S3Config, error) {
	tenant := ctxkeys.GetTenant(ctx)
	if tenant == nil {
		return nil, fmt.Errorf("tenant information not found in context")
	}

	// Only use tenant's own S3 config
	if tenant.S3Config.AccessKey == "" || tenant.S3Config.SecretKey == "" || tenant.S3Config.Bucket == "" {
		return nil, fmt.Errorf("S3 credentials are not configured for this tenant")
	}

	return &S3Config{
		Region:    tenant.S3Config.Region,
		Bucket:    tenant.S3Config.Bucket,
		AccessKey: tenant.S3Config.AccessKey,
		SecretKey: tenant.S3Config.SecretKey,
		Endpoint:  tenant.S3Config.Endpoint,
		UseSSL:    tenant.S3Config.UseSSL,
		PathStyle: tenant.S3Config.PathStyle,
		UserName:  tenant.S3Config.UserName,
	}, nil
}

// UploadFile uploads a file to S3 and returns the storage key
func (s *S3Service) UploadFile(ctx context.Context, fileContent io.Reader, originalName, contentType string) (string, error) {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		utils.Logger.Error("Failed to get tenant S3 config for upload",
			zap.Error(err),
			zap.String("filename", originalName))
		return "", fmt.Errorf("failed to get S3 config: %w", err)
	}

	// 🔍 [DEBUG] Логируем конфигурацию S3 (без секретов)
	utils.Logger.Info("S3 upload configuration",
		zap.String("filename", originalName),
		zap.String("bucket", config.Bucket),
		zap.String("region", config.Region),
		zap.String("endpoint", config.Endpoint),
		zap.Bool("use_ssl", config.UseSSL),
		zap.String("path_style", config.PathStyle),
		zap.Bool("has_access_key", config.AccessKey != ""),
		zap.Bool("has_secret_key", config.SecretKey != ""))

	client, err := s.getS3Client(config)
	if err != nil {
		utils.Logger.Error("Failed to create S3 client for upload",
			zap.Error(err),
			zap.String("filename", originalName),
			zap.String("bucket", config.Bucket),
			zap.String("endpoint", config.Endpoint))
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Generate unique storage key
	storageKey := s.generateStorageKey(originalName)

	// Create uploader
	uploader := s3manager.NewUploaderWithClient(client)

	utils.Logger.Info("Starting S3 upload",
		zap.String("filename", originalName),
		zap.String("storage_key", storageKey),
		zap.String("content_type", contentType))

	// Upload file
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(config.Bucket),
		Key:         aws.String(storageKey),
		Body:        fileContent,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		utils.Logger.Error("S3 upload operation failed",
			zap.Error(err),
			zap.String("filename", originalName),
			zap.String("storage_key", storageKey),
			zap.String("bucket", config.Bucket),
			zap.String("endpoint", config.Endpoint))
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	utils.Logger.Info("S3 upload completed successfully",
		zap.String("filename", originalName),
		zap.String("storage_key", storageKey),
		zap.String("s3_location", result.Location))

	return storageKey, nil
}

// DeleteFile deletes a file from S3
func (s *S3Service) DeleteFile(ctx context.Context, storageKey string) error {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		return fmt.Errorf("failed to get S3 config: %w", err)
	}

	client, err := s.getS3Client(config)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	_, err = client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetPresignedURL generates a presigned URL for file access
func (s *S3Service) GetPresignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error) {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get S3 config: %w", err)
	}

	client, err := s.getS3Client(config)
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

	req, _ := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(storageKey),
	})

	url, err := req.Presign(expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

// generateStorageKey generates a unique storage key for the file
func (s *S3Service) generateStorageKey(originalName string) string {
	ext := filepath.Ext(originalName)
	filename := strings.TrimSuffix(originalName, ext)

	// Sanitize filename
	filename = sanitizeFilename(filename)

	// Generate unique key components
	timestamp := time.Now().Format("2006/01/02")
	id := uuid.New().String()[:8] // Используем только первые 8 символов UUID

	// Calculate space available for filename
	// Format: timestamp/filename-id.ext
	// Example: 2024/01/15/filename-a1b2c3d4.pdf
	baseLength := len(timestamp) + 1 + 1 + len(id) + len(ext) // +1 для '/' и '-'
	maxFilenameLength := 1000 - baseLength                    // Оставляем запас в 24 символа для безопасности

	// Truncate filename if too long
	if len(filename) > maxFilenameLength {
		filename = truncateFilename(filename, maxFilenameLength)
	}

	storageKey := fmt.Sprintf("%s/%s-%s%s", timestamp, filename, id, ext)

	// Final safety check - should never happen but better safe than sorry
	if len(storageKey) > 1024 {
		// Emergency fallback - use only UUID and extension
		storageKey = fmt.Sprintf("%s/%s%s", timestamp, uuid.New().String(), ext)
	}

	return storageKey
}

// sanitizeFilename removes or replaces invalid characters from filename for S3 storage key
// This creates ASCII-safe keys while the original filename is preserved separately for display
func sanitizeFilename(filename string) string {
	if filename == "" {
		return "file"
	}

	// Remove extension for processing
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Use existing utility function for transliteration and sanitization
	sanitized := utils.GenerateCodeFromString(nameWithoutExt)

	// If result is empty or too generic, create a meaningful name
	if sanitized == "" || strings.HasPrefix(sanitized, "code_") {
		sanitized = "file"
	}

	return sanitized
}

// truncateFilename truncates filename to maxLength while trying to preserve readability
func truncateFilename(filename string, maxLength int) string {
	if len(filename) <= maxLength {
		return filename
	}

	// Try to truncate at word boundary (underscore or dash) near the end
	if maxLength > 10 {
		// Look for word boundaries from maxLength going backwards
		for i := maxLength - 1; i >= maxLength-10 && i > 0; i-- {
			if filename[i] == '_' || filename[i] == '-' {
				return filename[:i]
			}
		}
	}

	// If no good break point found, just truncate
	return filename[:maxLength]
}

// GetFileInfo returns information about a file in S3
func (s *S3Service) GetFileInfo(ctx context.Context, storageKey string) (*s3.HeadObjectOutput, error) {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 config: %w", err)
	}

	client, err := s.getS3Client(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	result, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return result, nil
}

// GetFileObject получает файл из S3 как поток для чтения
func (s *S3Service) GetFileObject(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 config: %w", err)
	}

	client, err := s.getS3Client(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	result, err := client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file object: %w", err)
	}

	return result.Body, nil
}

// CheckStorageLimit проверяет, не превысит ли загрузка файла лимит хранилища (с учетом буфера 10%)
func (s *S3Service) CheckStorageLimit(ctx context.Context, fileSize int64) error {
	// Получаем информацию о тенанте
	tenant := ctxkeys.GetTenant(ctx)
	if tenant == nil {
		return fmt.Errorf("tenant information not found in context")
	}

	// Получаем лимит хранилища
	storageLimit := tenant.S3Config.StorageLimitBytes
	if storageLimit < 0 {
		// Если лимит отрицательный, пропускаем проверку (не настроен)
		return nil
	}

	// Если лимит равен 0, блокируем любую загрузку
	if storageLimit == 0 {
		utils.Logger.Warn("Storage limit is zero - no uploads allowed",
			zap.String("tenant_id", tenant.ID.String()),
			zap.Int64("file_size", fileSize),
		)

		return fmt.Errorf("%s", utils.T(ctx, "error.file.storage_not_configured"))
	}

	// Добавляем буфер 10%
	bufferLimit := int64(float64(storageLimit) * 1.1)

	// Используем текущее использование из контекста тенанта
	currentUsage := tenant.S3Config.CurrentUsageBytes

	// Проверяем, не превысит ли новый файл лимит с буфером
	if currentUsage+fileSize > bufferLimit {
		storageLimitGB := storageLimit / (1024 * 1024 * 1024)
		currentUsageGB := currentUsage / (1024 * 1024 * 1024)

		utils.Logger.Warn("Storage limit exceeded",
			zap.String("tenant_id", tenant.ID.String()),
			zap.Int64("current_usage_bytes", currentUsage),
			zap.Int64("current_usage_gb", currentUsageGB),
			zap.Int64("storage_limit_bytes", storageLimit),
			zap.Int64("storage_limit_gb", storageLimitGB),
			zap.Int64("file_size", fileSize),
			zap.Int64("buffer_limit_bytes", bufferLimit),
		)

		return fmt.Errorf("%s", utils.T(ctx, "error.file.storage_limit_exceeded", map[string]interface{}{
			"current_usage_gb": currentUsageGB,
			"limit_gb":         storageLimitGB,
		}))
	}

	return nil
}

// CheckStorageLimitWithFilename проверяет лимит хранилища с возможностью аудита (для использования в FileService)
func (s *S3Service) CheckStorageLimitWithFilename(ctx context.Context, fileName string, fileSize int64) error {
	// Получаем информацию о тенанте
	tenant := ctxkeys.GetTenant(ctx)
	if tenant == nil {
		utils.Logger.Error("Tenant information not found in context for storage limit check",
			zap.String("file_name", fileName),
			zap.Int64("file_size", fileSize))
		return fmt.Errorf("tenant information not found in context")
	}

	utils.Logger.Info("Checking storage limit",
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("file_name", fileName),
		zap.Int64("file_size", fileSize),
		zap.Int64("storage_limit", tenant.S3Config.StorageLimitBytes),
		zap.Int64("current_usage", tenant.S3Config.CurrentUsageBytes))

	// Получаем лимит хранилища
	storageLimit := tenant.S3Config.StorageLimitBytes
	if storageLimit < 0 {
		utils.Logger.Info("Storage limit is negative - skipping check",
			zap.String("tenant_id", tenant.ID.String()),
			zap.Int64("storage_limit", storageLimit))
		// Если лимит отрицательный, пропускаем проверку (не настроен)
		return nil
	}

	// Если лимит равен 0, блокируем любую загрузку
	if storageLimit == 0 {
		utils.Logger.Warn("Storage limit is zero - no uploads allowed",
			zap.String("tenant_id", tenant.ID.String()),
			zap.String("file_name", fileName),
			zap.Int64("file_size", fileSize),
		)

		// Возвращаем специальную ошибку для незастроенного хранилища
		return &StorageNotConfiguredError{
			FileName: fileName,
			FileSize: fileSize,
		}
	}

	// Используем текущее использование из контекста тенанта
	currentUsage := tenant.S3Config.CurrentUsageBytes

	// Определяем подходящие единицы для лимита (используем везде)
	var limit64, limitUnit string
	if storageLimit >= 1024*1024*1024 {
		limit64 = fmt.Sprintf("%.1f", float64(storageLimit)/(1024*1024*1024))
		limitUnit = utils.T(ctx, "units.storage.gb")
	} else {
		limit64 = fmt.Sprintf("%.0f", float64(storageLimit)/(1024*1024))
		limitUnit = utils.T(ctx, "units.storage.mb")
	}

	// Сначала проверяем, не больше ли файл сам по себе лимита (когда ничего не загружено)
	if currentUsage == 0 && fileSize > storageLimit {
		// Определяем единицы для размера файла
		var fileSize64, fileUnit string
		if fileSize >= 1024*1024*1024 {
			fileSize64 = fmt.Sprintf("%.1f", float64(fileSize)/(1024*1024*1024))
			fileUnit = utils.T(ctx, "units.storage.gb")
		} else {
			fileSize64 = fmt.Sprintf("%.0f", float64(fileSize)/(1024*1024))
			fileUnit = utils.T(ctx, "units.storage.mb")
		}

		utils.Logger.Warn("File too large for storage limit",
			zap.String("tenant_id", tenant.ID.String()),
			zap.String("file_name", fileName),
			zap.Int64("file_size", fileSize),
			zap.Int64("storage_limit", storageLimit),
		)

		return &FileTooLargeError{
			FileName:   fileName,
			FileSize:   fileSize,
			FileSize64: fileSize64,
			FileUnit:   fileUnit,
			Limit64:    limit64,
			LimitUnit:  limitUnit,
		}
	}

	// Добавляем буфер 10%
	bufferLimit := int64(float64(storageLimit) * 1.1)

	// Проверяем, не превысит ли новый файл лимит с буфером
	if currentUsage+fileSize > bufferLimit {
		// Определяем подходящие единицы для текущего использования
		var currentUsage64, currentUnit string
		if currentUsage >= 1024*1024*1024 {
			currentUsage64 = fmt.Sprintf("%.1f", float64(currentUsage)/(1024*1024*1024))
			currentUnit = utils.T(ctx, "units.storage.gb")
		} else {
			currentUsage64 = fmt.Sprintf("%.0f", float64(currentUsage)/(1024*1024))
			currentUnit = utils.T(ctx, "units.storage.mb")
		}

		utils.Logger.Warn("Storage limit exceeded",
			zap.String("tenant_id", tenant.ID.String()),
			zap.String("file_name", fileName),
			zap.Int64("current_usage_bytes", currentUsage),
			zap.Int64("storage_limit_bytes", storageLimit),
			zap.Int64("file_size", fileSize),
			zap.Int64("buffer_limit_bytes", bufferLimit),
		)

		// Возвращаем специальную ошибку с данными для аудита
		return &StorageLimitError{
			FileName:       fileName,
			FileSize:       fileSize,
			CurrentUsage:   currentUsage,
			StorageLimit:   storageLimit,
			CurrentUsage64: currentUsage64,
			CurrentUnit:    currentUnit,
			Limit64:        limit64,
			LimitUnit:      limitUnit,
		}
	}

	return nil
}

// StorageLimitError представляет ошибку превышения лимита хранилища с данными для аудита
type StorageLimitError struct {
	FileName       string
	FileSize       int64
	CurrentUsage   int64
	StorageLimit   int64
	CurrentUsage64 string
	CurrentUnit    string
	Limit64        string
	LimitUnit      string
}

func (e *StorageLimitError) Error() string {
	return fmt.Sprintf("storage limit exceeded: current usage %s %s, limit %s %s",
		e.CurrentUsage64, e.CurrentUnit, e.Limit64, e.LimitUnit)
}

// StorageNotConfiguredError представляет ошибку для незастроенного хранилища
type StorageNotConfiguredError struct {
	FileName string
	FileSize int64
}

func (e *StorageNotConfiguredError) Error() string {
	return fmt.Sprintf("storage limit is not configured for this file: %s, size %d bytes",
		e.FileName, e.FileSize)
}

// FileTooLargeError представляет ошибку когда файл сам по себе больше лимита хранилища
type FileTooLargeError struct {
	FileName   string
	FileSize   int64
	FileSize64 string
	FileUnit   string
	Limit64    string
	LimitUnit  string
}

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("file size %s %s exceeds storage limit %s %s",
		e.FileSize64, e.FileUnit, e.Limit64, e.LimitUnit)
}

// UploadTemporaryFile uploads a temporary file to S3 with a custom storage key
func (s *S3Service) UploadTemporaryFile(ctx context.Context, fileContent io.Reader, storageKey, contentType string) error {
	config, err := s.getTenantS3Config(ctx)
	if err != nil {
		return fmt.Errorf("failed to get S3 config: %w", err)
	}

	client, err := s.getS3Client(config)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create uploader
	uploader := s3manager.NewUploaderWithClient(client)

	// Upload file with custom key
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(config.Bucket),
		Key:         aws.String(storageKey),
		Body:        fileContent,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload temporary file: %w", err)
	}

	return nil
}
