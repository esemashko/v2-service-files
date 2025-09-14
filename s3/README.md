# S3 Service Package

This package provides S3-compatible storage service for multi-tenant file management.

## Overview

The S3 service manages file storage operations with tenant isolation, ensuring that each tenant's files are stored in separate prefixes within a shared S3 bucket.

## Configuration

The service is configured through environment variables:

```bash
# S3 Connection Settings
S3_REGION=us-east-1                    # AWS region (default: us-east-1)
S3_BUCKET=your-bucket-name             # S3 bucket name (required)
S3_ACCESS_KEY=your-access-key          # S3 access key (required)
S3_SECRET_KEY=your-secret-key          # S3 secret key (required)
S3_ENDPOINT=https://s3.amazonaws.com   # S3 endpoint URL (optional, for MinIO/custom S3)
S3_USE_SSL=true                        # Use SSL for connections (default: true)
S3_PATH_STYLE=auto                     # Path style: auto, path, or virtual (default: auto)

# Storage Limits
S3_STORAGE_LIMIT_BYTES=-1              # Storage limit per tenant in bytes (-1 = unlimited)
```

### Configuration Examples

#### AWS S3
```bash
S3_REGION=us-west-2
S3_BUCKET=my-app-files
S3_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_STORAGE_LIMIT_BYTES=10737418240  # 10GB
```

#### MinIO (Local Development)
```bash
S3_REGION=us-east-1
S3_BUCKET=dev-files
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_ENDPOINT=http://localhost:9000
S3_USE_SSL=false
S3_PATH_STYLE=path
S3_STORAGE_LIMIT_BYTES=1073741824  # 1GB
```

#### DigitalOcean Spaces
```bash
S3_REGION=nyc3
S3_BUCKET=my-space-name
S3_ACCESS_KEY=DO_SPACES_KEY
S3_SECRET_KEY=DO_SPACES_SECRET
S3_ENDPOINT=https://nyc3.digitaloceanspaces.com
S3_USE_SSL=true
S3_PATH_STYLE=virtual
```

## Tenant Isolation

Files are automatically organized by tenant using the following structure:
```
bucket/
├── tenants/
│   ├── {tenant-id-1}/
│   │   ├── 2024/01/01/file1-abc123.pdf
│   │   └── 2024/01/02/file2-def456.jpg
│   └── {tenant-id-2}/
│       └── 2024/01/01/file3-ghi789.docx
```

The tenant ID is automatically extracted from the federation context and used as a prefix for all operations.

## Features

### File Operations
- **Upload**: Upload files with automatic tenant prefixing
- **Delete**: Remove files from S3
- **Download**: Get file objects as streams
- **Presigned URLs**: Generate temporary access URLs
- **File Info**: Retrieve file metadata

### Storage Management
- **Storage Limits**: Configurable per-tenant storage limits
- **Usage Tracking**: Monitor storage usage (requires external tracking)
- **Buffer Zone**: 10% buffer on storage limits to prevent hard stops
- **Detailed Error Messages**: Localized error messages for storage issues

### Security
- **Tenant Isolation**: Files are strictly separated by tenant ID
- **Presigned URLs**: Time-limited access to files
- **Sanitized Keys**: File names are sanitized for S3 compatibility

## Usage

### Initialize Service
```go
s3Service := s3.NewS3Service()
```

### Upload File
```go
storageKey, err := s3Service.UploadFile(ctx, fileReader, "document.pdf", "application/pdf")
if err != nil {
    // Handle error
}
```

### Generate Presigned URL
```go
url, err := s3Service.GetPresignedURL(ctx, storageKey, 1*time.Hour)
if err != nil {
    // Handle error
}
```

### Check Storage Limit
```go
// currentUsage should be fetched from database
err := s3Service.CheckStorageLimit(ctx, fileSize, currentUsage)
if err != nil {
    switch e := err.(type) {
    case *s3.StorageLimitError:
        // Storage limit exceeded
    case *s3.StorageNotConfiguredError:
        // Storage not configured
    case *s3.FileTooLargeError:
        // File too large for limit
    }
}
```

### Delete File
```go
err := s3Service.DeleteFile(ctx, storageKey)
if err != nil {
    // Handle error
}
```

## Error Types

The service provides typed errors for better error handling:

- `StorageLimitError`: Storage limit exceeded
- `StorageNotConfiguredError`: Storage limit is set to 0
- `FileTooLargeError`: Single file exceeds total storage limit

## Storage Key Format

Files are stored with the following key structure:
```
tenants/{tenant-id}/{year}/{month}/{day}/{sanitized-filename}-{unique-id}.{extension}
```

Example:
```
tenants/550e8400-e29b-41d4-a716-446655440000/2024/01/15/invoice-a1b2c3d4.pdf
```

## Migration from Per-Tenant Configuration

This service has been updated to use a shared S3 configuration for all tenants instead of per-tenant S3 credentials. The main changes:

1. **Single S3 Configuration**: All tenants share the same S3 bucket and credentials
2. **Tenant Prefixing**: Files are automatically prefixed with `tenants/{tenant-id}/`
3. **Federation Context**: Tenant ID is retrieved from federation context instead of ctxkeys
4. **Environment Configuration**: All settings come from environment variables

## Testing

### Local Testing with MinIO

1. Start MinIO:
```bash
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"
```

2. Create bucket:
```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/test-bucket
```

3. Configure environment:
```bash
export S3_BUCKET=test-bucket
export S3_ACCESS_KEY=minioadmin
export S3_SECRET_KEY=minioadmin
export S3_ENDPOINT=http://localhost:9000
export S3_USE_SSL=false
export S3_PATH_STYLE=path
```

## Performance Considerations

- Files are streamed directly to S3 without buffering in memory
- Presigned URLs avoid proxying files through the application
- Storage limit checks should cache current usage to avoid frequent DB queries

## Security Notes

- Never expose S3 credentials in logs or error messages
- Use presigned URLs with appropriate expiration times
- Validate file types and sizes before upload
- Ensure proper context validation before any S3 operation