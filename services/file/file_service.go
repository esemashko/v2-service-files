package file

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"main/ctxkeys"
	"main/ent"
	"main/ent/file"
	"main/ent/user"
	fileprivacy "main/privacy/file"
	"main/s3"
	"main/types"
	"main/utils"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// DefaultPresignedURLExpiration время жизни pre-signed URL по умолчанию (1 час)
	DefaultPresignedURLExpiration = time.Hour
	// MaxPresignedURLExpiration максимальное время жизни pre-signed URL (24 часа)
	MaxPresignedURLExpiration = 24 * time.Hour
	// MaxBatchArchiveFiles максимальное количество файлов в архиве
	MaxBatchArchiveFiles = 50
)

// FileService provides file management operations
type FileService struct {
	s3Service    *s3.S3Service
	auditService *FileAuditService
}

// hasAdminRole проверяет, имеет ли пользователь админскую роль
func (s *FileService) hasAdminRole(ctx context.Context) bool {
	localUser := ctxkeys.GetLocalUser(ctx)
	if localUser == nil {
		return false
	}

	userRole, err := localUser.Role(ctx)
	if err != nil {
		return false
	}

	return userRole != nil && types.IsRoleHigherOrEqual(userRole.Code, types.RoleAdmin)
}

// isMember проверяет, имеет ли пользователь роль member или выше
func (s *FileService) isMember(ctx context.Context) bool {
	localUser := ctxkeys.GetLocalUser(ctx)
	if localUser == nil {
		return false
	}

	userRole, err := localUser.Role(ctx)
	if err != nil {
		return false
	}

	return userRole != nil && types.IsRoleHigherOrEqual(userRole.Code, types.RoleMember)
}

// canDownloadFile проверяет, может ли пользователь скачивать файл
func (s *FileService) canDownloadFile(ctx context.Context, client *ent.Client, fileID uuid.UUID) error {
	// Убедимся, что файл существует
	if _, err := client.File.Query().
		Where(file.ID(fileID)).
		Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	// Аутентификация пользователя и роль
	userID := ctxkeys.GetUserID(ctx)
	var userRoleCode string
	if localUser := ctxkeys.GetLocalUser(ctx); localUser != nil {
		if userRole, err := localUser.Role(ctx); err == nil && userRole != nil {
			userRoleCode = userRole.Code
		}
	}

	// Проверяем доступ через централизованные предикаты (автор файла, доступ к тикету/комментарию/чату)
	if err := fileprivacy.CanAccessFile(ctx, client, userID, userRoleCode, fileID); err != nil {
		return fmt.Errorf("%s", utils.T(ctx, "error.file.view_permission_denied"))
	}
	return nil
}

// CanUpdateFile проверяет, может ли пользователь редактировать файл
func (s *FileService) CanUpdateFile(ctx context.Context, client *ent.Client, fileID uuid.UUID) error {
	userID := ctxkeys.GetUserID(ctx)

	// Получаем файл с загрузчиком
	fileRecord, err := client.File.Query().
		Where(file.ID(fileID)).
		WithUploader().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	// Владельцы и администраторы могут редактировать любые файлы
	if s.hasAdminRole(ctx) {
		return nil
	}

	// Пользователи могут редактировать только свои файлы
	if fileRecord.Edges.Uploader != nil && fileRecord.Edges.Uploader.ID == userID {
		return nil
	}

	return fmt.Errorf("%s", utils.T(ctx, "error.file.update_permission_denied"))
}

// CanUploadFile проверяет, может ли пользователь загружать файлы
func (s *FileService) CanUploadFile(ctx context.Context) error {
	userID := ctxkeys.GetUserID(ctx)
	if userID != uuid.Nil {
		return nil
	}
	return fmt.Errorf("%s", utils.T(ctx, "error.file.upload_permission_denied"))
}

// CanDeleteFile проверяет, может ли пользователь удалять файл
func (s *FileService) CanDeleteFile(ctx context.Context, client *ent.Client, fileID uuid.UUID) error {
	userID := ctxkeys.GetUserID(ctx)

	// Получаем файл с загрузчиком
	fileRecord, err := client.File.Query().
		Where(file.ID(fileID)).
		WithUploader().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	// Владельцы и администраторы могут удалять любые файлы
	if s.hasAdminRole(ctx) {
		return nil
	}

	// Пользователи могут удалять только свои файлы
	if fileRecord.Edges.Uploader != nil && fileRecord.Edges.Uploader.ID == userID {
		return nil
	}

	return fmt.Errorf("%s", utils.T(ctx, "error.file.delete_permission_denied"))
}

// CanViewFile проверяет, может ли пользователь просматривать файл
func (s *FileService) CanViewFile(ctx context.Context, client *ent.Client, fileID uuid.UUID) error {
	// Логика такая же, как для скачивания файла
	return s.canDownloadFile(ctx, client, fileID)
}

// removed: GetFilePermissions — deprecated in favor of field-level canDelete

// NewFileService creates a new file service
func NewFileService() *FileService {
	return &FileService{
		s3Service:    s3.NewS3Service(),
		auditService: NewFileAuditService(),
	}
}

// UploadFileInput contains file upload parameters
type UploadFileInput struct {
	Upload      *graphql.Upload
	Description *string
}

// FileDownloadUrlResult содержит данные о pre-signed URL для скачивания файла
type FileDownloadUrlResult struct {
	URL       string
	ExpiresAt time.Time
}

// BatchDownloadUrlResult содержит данные о pre-signed URL для скачивания архива
type BatchDownloadUrlResult struct {
	URL         string
	ExpiresAt   time.Time
	ArchiveName string
	TotalFiles  int
}

// GetFileDownloadURL генерирует pre-signed URL для скачивания одиночного файла
func (s *FileService) GetFileDownloadURL(ctx context.Context, client *ent.Client, fileID uuid.UUID) (*FileDownloadUrlResult, error) {
	// 🔒 [POLICY CHECK] Проверяем права на скачивание файла
	if err := s.canDownloadFile(ctx, client, fileID); err != nil {
		return nil, err
	}

	// Получаем файл из базы данных
	fileRecord, err := client.File.Query().
		Where(file.ID(fileID)).
		WithUploader().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	// Генерируем pre-signed URL с временем жизни 1 час
	url, err := s.s3Service.GetPresignedURL(ctx, fileRecord.StorageKey, DefaultPresignedURLExpiration)
	if err != nil {
		if strings.Contains(err.Error(), "S3 credentials are not configured") {
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.s3_not_configured"))
		}
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.url_generation_failed"))
	}

	// 📊 [AUDIT] Логируем генерацию URL для скачивания
	s.auditService.LogFileDownloadUrlGeneration(ctx, client, fileID)

	return &FileDownloadUrlResult{
		URL:       url,
		ExpiresAt: time.Now().Add(DefaultPresignedURLExpiration),
	}, nil
}

// GetBatchDownloadURL создает ZIP архив из указанных файлов и возвращает pre-signed URL для его скачивания
func (s *FileService) GetBatchDownloadURL(ctx context.Context, client *ent.Client, fileIDs []uuid.UUID, archiveName string) (*BatchDownloadUrlResult, error) {
	// Валидация входных данных
	if len(fileIDs) == 0 {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.no_files_selected"))
	}
	if len(fileIDs) > MaxBatchArchiveFiles {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.too_many_files_selected"))
	}

	// Получаем и проверяем права на все файлы
	files, err := s.validateAndGetFilesForBatch(ctx, client, fileIDs)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.no_accessible_files"))
	}

	// Генерируем имя архива, если не задано
	if archiveName == "" {
		archiveName = fmt.Sprintf("files_%s.zip", time.Now().Format("20060102_150405"))
	}
	if !strings.HasSuffix(archiveName, ".zip") {
		archiveName += ".zip"
	}

	// Создаем ZIP архив в памяти
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	usedFilenames := make(map[string]bool)

	for _, fileRecord := range files {
		if err := s.addFileToZipFromS3(ctx, zipWriter, fileRecord, usedFilenames); err != nil {
			utils.Logger.Error("Failed to add file to ZIP archive",
				zap.Error(err),
				zap.String("file_id", fileRecord.ID.String()),
				zap.String("filename", fileRecord.OriginalName))
			// Продолжаем обработку других файлов
			continue
		}

		// 📊 [AUDIT] Логируем каждый файл отдельно как скачанный в составе архива
		s.auditService.LogFileBatchDownload(ctx, client, fileRecord.ID, archiveName, len(files))
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.archive_creation_failed"))
	}

	// Загружаем архив в S3 с временным ключом
	archiveStorageKey := s.generateTemporaryArchiveKey(archiveName)
	err = s.s3Service.UploadTemporaryFile(ctx, &buffer, archiveStorageKey, "application/zip")
	if err != nil {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.archive_upload_failed"))
	}

	// Генерируем pre-signed URL для архива
	url, err := s.s3Service.GetPresignedURL(ctx, archiveStorageKey, DefaultPresignedURLExpiration)
	if err != nil {
		// Удаляем архив при ошибке генерации URL
		_ = s.s3Service.DeleteFile(ctx, archiveStorageKey)
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.url_generation_failed"))
	}

	// Планируем удаление архива через 1 час
	go s.scheduleArchiveDeletion(ctx, archiveStorageKey, DefaultPresignedURLExpiration)

	utils.Logger.Info("Batch download archive created",
		zap.Int("total_files", len(files)),
		zap.Int("requested_files", len(fileIDs)),
		zap.String("archive_name", archiveName),
		zap.String("storage_key", archiveStorageKey))

	return &BatchDownloadUrlResult{
		URL:         url,
		ExpiresAt:   time.Now().Add(DefaultPresignedURLExpiration),
		ArchiveName: archiveName,
		TotalFiles:  len(files),
	}, nil
}

// validateAndGetFilesForBatch проверяет права доступа и получает файлы для группового скачивания
func (s *FileService) validateAndGetFilesForBatch(ctx context.Context, client *ent.Client, fileIDs []uuid.UUID) ([]*ent.File, error) {
	// Получаем все файлы из базы данных
	files, err := client.File.Query().
		Where(file.IDIn(fileIDs...)).
		WithUploader().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.get_files_failed"))
	}

	// Проверяем права на каждый файл
	var accessibleFiles []*ent.File
	for _, fileRecord := range files {
		if err := s.canDownloadFile(ctx, client, fileRecord.ID); err != nil {
			utils.Logger.Warn("File access denied in batch download",
				zap.String("file_id", fileRecord.ID.String()),
				zap.Error(err))
			// Пропускаем файлы без доступа, но не фейлим весь запрос
			continue
		}
		accessibleFiles = append(accessibleFiles, fileRecord)
	}

	return accessibleFiles, nil
}

// addFileToZipFromS3 добавляет файл из S3 в ZIP-архив
func (s *FileService) addFileToZipFromS3(ctx context.Context, zipWriter *zip.Writer, fileRecord *ent.File, usedFilenames map[string]bool) error {
	// Получаем файл из S3
	s3Object, err := s.s3Service.GetFileObject(ctx, fileRecord.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to get file from S3: %w", err)
	}
	defer s3Object.Close()

	// Создаем уникальное имя файла в архиве
	filename := s.generateUniqueFilename(fileRecord.OriginalName, usedFilenames)

	// Создаем заголовок файла в ZIP
	header := &zip.FileHeader{
		Name:   filename,
		Method: zip.Store, // Без сжатия для скорости
	}
	header.SetModTime(fileRecord.CreateTime)

	// Создаем writer для файла в архиве
	fileWriter, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create file header in ZIP: %w", err)
	}

	// Копируем содержимое файла в архив
	written, err := io.Copy(fileWriter, s3Object)
	if err != nil {
		return fmt.Errorf("failed to write file to ZIP: %w", err)
	}

	utils.Logger.Debug("File added to ZIP archive",
		zap.String("file_id", fileRecord.ID.String()),
		zap.String("filename", filename),
		zap.Int64("size", written))

	return nil
}

// generateUniqueFilename создает уникальное имя файла для архива
func (s *FileService) generateUniqueFilename(originalName string, usedFilenames map[string]bool) string {
	if !usedFilenames[originalName] {
		usedFilenames[originalName] = true
		return originalName
	}

	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
		if !usedFilenames[newName] {
			usedFilenames[newName] = true
			return newName
		}
		counter++
	}
}

// generateTemporaryArchiveKey генерирует ключ для временного архива в корневой временной папке S3
func (s *FileService) generateTemporaryArchiveKey(archiveName string) string {
	timestamp := time.Now().Format("2006/01/02/15")
	id := uuid.New().String()[:8]

	// Сохраняем во временную папку в корне бакета
	return fmt.Sprintf("temp/%s/%s-%s", timestamp, strings.TrimSuffix(archiveName, ".zip"), id) + ".zip"
}

// scheduleArchiveDeletion планирует удаление временного архива через указанное время
func (s *FileService) scheduleArchiveDeletion(ctx context.Context, storageKey string, delay time.Duration) {
	// Ждем указанное время
	time.Sleep(delay)

	// Удаляем архив из S3
	if err := s.s3Service.DeleteFile(ctx, storageKey); err != nil {
		utils.Logger.Error("Failed to delete temporary archive",
			zap.Error(err),
			zap.String("storage_key", storageKey))
	} else {
		utils.Logger.Info("Temporary archive deleted successfully",
			zap.String("storage_key", storageKey))
	}
}

// UploadFile uploads a file to S3 and creates a file record in database
func (s *FileService) UploadFile(ctx context.Context, client *ent.Client, input UploadFileInput) (*ent.File, error) {
	utils.Logger.Info("UploadFile method called",
		zap.String("filename", input.Upload.Filename),
		zap.Int64("file_size", input.Upload.Size),
		zap.Bool("client_not_nil", client != nil))

	if input.Upload == nil {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.no_file"))
	}

	upload := input.Upload

	// Validate filename length (prevent S3 key length issues)
	if len(upload.Filename) > 200 {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.filename_too_long"))
	}

	// Validate file size (limit to 100MB)
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if upload.Size > maxFileSize {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.size_too_large"))
	}

	// Detect content type if not provided or empty
	contentType := upload.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(upload.Filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// 📊 [STORAGE LIMIT CHECK] Проверяем лимит хранилища перед загрузкой
	if err := s.s3Service.CheckStorageLimitWithFilename(ctx, upload.Filename, upload.Size); err != nil {
		utils.Logger.Info("Storage limit check failed",
			zap.String("filename", upload.Filename),
			zap.Int64("file_size", upload.Size),
			zap.Error(err))

		// Проверяем, является ли это ошибкой незастроенного хранилища
		if storageNotConfiguredErr, ok := err.(*s3.StorageNotConfiguredError); ok {
			utils.Logger.Info("Logging storage not configured violation",
				zap.String("filename", storageNotConfiguredErr.FileName),
				zap.Int64("file_size", storageNotConfiguredErr.FileSize))

			// Логируем попытку загрузки в незастроенное хранилище
			utils.Logger.Info("About to call LogStorageNotConfiguredViolation",
				zap.String("filename", storageNotConfiguredErr.FileName),
				zap.Int64("file_size", storageNotConfiguredErr.FileSize))

			s.auditService.LogStorageNotConfiguredViolation(ctx, client,
				storageNotConfiguredErr.FileName,
				storageNotConfiguredErr.FileSize)

			utils.Logger.Info("LogStorageNotConfiguredViolation call completed")

			// Возвращаем локализованную ошибку пользователю
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.storage_not_configured"))
		}

		// Проверяем, является ли это ошибкой превышения лимита с данными для аудита
		if storageLimitErr, ok := err.(*s3.StorageLimitError); ok {
			utils.Logger.Info("Logging storage limit violation",
				zap.String("filename", storageLimitErr.FileName),
				zap.Int64("file_size", storageLimitErr.FileSize),
				zap.Int64("current_usage", storageLimitErr.CurrentUsage),
				zap.Int64("storage_limit", storageLimitErr.StorageLimit))

			// Логируем попытку превышения лимита
			utils.Logger.Info("About to call LogStorageLimitViolation",
				zap.String("filename", storageLimitErr.FileName),
				zap.Int64("file_size", storageLimitErr.FileSize))

			s.auditService.LogStorageLimitViolation(ctx, client,
				storageLimitErr.FileName,
				storageLimitErr.FileSize,
				storageLimitErr.CurrentUsage,
				storageLimitErr.StorageLimit)

			utils.Logger.Info("LogStorageLimitViolation call completed")

			// Возвращаем локализованную ошибку пользователю
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.storage_limit_exceeded", map[string]interface{}{
				"current_usage": storageLimitErr.CurrentUsage64,
				"current_unit":  storageLimitErr.CurrentUnit,
				"limit":         storageLimitErr.Limit64,
				"limit_unit":    storageLimitErr.LimitUnit,
			}))
		}

		// Проверяем, является ли это ошибкой файла, который сам по себе больше лимита
		if fileTooLargeErr, ok := err.(*s3.FileTooLargeError); ok {
			utils.Logger.Info("File too large for storage limit",
				zap.String("filename", fileTooLargeErr.FileName),
				zap.Int64("file_size", fileTooLargeErr.FileSize))

			// Возвращаем локализованную ошибку пользователю
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.file_too_large_for_storage", map[string]interface{}{
				"file_size":  fileTooLargeErr.FileSize64,
				"file_unit":  fileTooLargeErr.FileUnit,
				"limit":      fileTooLargeErr.Limit64,
				"limit_unit": fileTooLargeErr.LimitUnit,
			}))
		}
		return nil, err
	}

	// Upload to S3
	storageKey, err := s.s3Service.UploadFile(ctx, upload.File, upload.Filename, contentType)
	if err != nil {
		// 🔍 [DEBUG] Логируем детальную ошибку S3 для диагностики
		utils.Logger.Error("S3 upload failed - detailed error",
			zap.Error(err),
			zap.String("filename", upload.Filename),
			zap.String("content_type", contentType),
			zap.Int64("file_size", upload.Size))

		// Check if it's S3 configuration error
		if strings.Contains(err.Error(), "S3 credentials are not configured") {
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.s3_not_configured"))
		}

		// Check for timeout errors
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			utils.Logger.Error("S3 upload timeout detected",
				zap.Error(err),
				zap.String("filename", upload.Filename))
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.upload_timeout"))
		}

		// Check for connection errors
		if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "network") {
			utils.Logger.Error("S3 connection error detected",
				zap.Error(err),
				zap.String("filename", upload.Filename))
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.s3_connection_failed"))
		}

		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.upload_failed"))
	}

	// Get user from context for database record
	localUser := ctxkeys.GetLocalUser(ctx)
	if localUser == nil {
		// Cleanup S3 file if user not found
		if deleteErr := s.s3Service.DeleteFile(ctx, storageKey); deleteErr != nil {
			utils.Logger.Error("Failed to cleanup S3 file after user context error",
				zap.Error(deleteErr),
				zap.String("storage_key", storageKey),
			)
		}
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.user.not_authenticated"))
	}

	// Create file record in database
	ctxWithClient := ent.NewContext(ctx, client)
	fileRecord, err := client.File.Create().
		SetOriginalName(upload.Filename).
		SetStorageKey(storageKey).
		SetMimeType(contentType).
		SetSize(upload.Size).
		SetUploaderID(localUser.ID).
		SetNillableDescription(input.Description).
		Save(ctxWithClient)
	if err != nil {
		// If database save fails, try to cleanup S3 file
		if deleteErr := s.s3Service.DeleteFile(ctx, storageKey); deleteErr != nil {
			utils.Logger.Error("Failed to cleanup S3 file after database error",
				zap.Error(deleteErr),
				zap.String("storage_key", storageKey),
			)
		}
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.create_failed"))
	}

	return fileRecord, nil
}

// DeleteFile deletes a file from both database and S3
func (s *FileService) DeleteFile(ctx context.Context, client *ent.Client, fileID uuid.UUID) error {
	ctxWithClient := ent.NewContext(ctx, client)

	// Проверяем существование файла перед удалением
	_, err := client.File.Query().
		Where(file.ID(fileID)).
		Only(ctxWithClient)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	// Жестко удаляем файл из базы данных
	err = client.File.DeleteOneID(fileID).
		Exec(ctxWithClient)
	if err != nil {
		return fmt.Errorf("%s", utils.T(ctx, "error.file.delete_failed"))
	}

	// Delete from S3 происходит автоматически через хук WithFileS3Deletion()

	return nil
}

// GetFilesByUser returns files uploaded by a specific user
func (s *FileService) GetFilesByUser(ctx context.Context, client *ent.Client, userID uuid.UUID, limit, offset int) ([]*ent.File, error) {
	ctxWithClient := ent.NewContext(ctx, client)

	files, err := client.File.Query().
		Where(file.HasUploaderWith(user.ID(userID))).
		Limit(limit).
		Offset(offset).
		Order(ent.Desc(file.FieldCreateTime)).
		All(ctxWithClient)
	if err != nil {
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.get_files_failed"))
	}

	return files, nil
}

// GetFileInfo returns file information
func (s *FileService) GetFileInfo(ctx context.Context, client *ent.Client, fileID uuid.UUID) (*ent.File, error) {
	ctxWithClient := ent.NewContext(ctx, client)

	fileRecord, err := client.File.Query().
		Where(file.ID(fileID)).
		WithUploader().
		Only(ctxWithClient)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.not_found"))
		}
		return nil, fmt.Errorf("%s", utils.T(ctx, "error.file.get_failed"))
	}

	return fileRecord, nil
}

// UpdateFilesBatch: visibility removed, method retained to avoid breaking callers until resolvers are cleaned
func (s *FileService) UpdateFilesBatch(ctx context.Context, client *ent.Client, fileIDs []uuid.UUID) ([]*ent.File, int, error) {
	// Валидация входных данных
	if len(fileIDs) == 0 {
		return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.no_files_selected"))
	}

	// Ограничиваем количество файлов для обновления за раз
	const maxBatchUpdateFiles = 100
	if len(fileIDs) > maxBatchUpdateFiles {
		return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.too_many_files_for_batch_update"))
	}

	// Проверяем права на все файлы перед началом обновления
	for _, fileID := range fileIDs {
		if err := s.CanUpdateFile(ctx, client, fileID); err != nil {
			return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.access_denied_for_batch_update"))
		}
	}

	// Получаем все файлы из базы данных для проверки их существования
	ctxWithClient := ent.NewContext(ctx, client)
	files, err := client.File.Query().
		Where(file.IDIn(fileIDs...)).
		WithUploader().
		Limit(maxBatchUpdateFiles).
		All(ctxWithClient)
	if err != nil {
		return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.get_files_failed"))
	}

	// Проверяем, что все файлы найдены
	if len(files) != len(fileIDs) {
		return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.some_files_not_found"))
	}

	// Возвращаем найденные файлы без изменения полей
	updatedCount := len(files)
	updatedFilesWithDetails, err := client.File.Query().
		Where(file.IDIn(fileIDs...)).
		WithUploader().
		Limit(maxBatchUpdateFiles).
		All(ctxWithClient)
	if err != nil {
		return nil, 0, fmt.Errorf("%s", utils.T(ctx, "error.file.get_updated_files_failed"))
	}

	return updatedFilesWithDetails, updatedCount, nil
}
