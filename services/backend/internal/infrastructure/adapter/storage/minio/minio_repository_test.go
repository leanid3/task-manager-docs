package minio

import (
	"app/config"
	apperrors "app/internal/entity/errors"
	appminio "app/pkg/minio"
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/require"
	"github.com/google/uuid"
	tminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func setupMinio(t *testing.T) (*MinioAdapter, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tminio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	require.NoError(t, err)

	// ConnectionString возвращает строку вида "host:port"
	connectionString, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	// MinioContainer имеет поля Username и Password напрямую
	accessKey := container.Username
	secretKey := container.Password

	cfg := &config.MinioConfig{
		Endpoint:  connectionString,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		Timeout:   5 * time.Second,
	}

	connector, err := appminio.NewConnector(cfg, NewMockLogger())
	require.NoError(t, err)

	adapter := NewMinioAdapter(connector, NewMockLogger())

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	return adapter, cleanup
}

func TestMinioAdapterUploadMetadataAndExists(t *testing.T) {
	adapter, cleanup := setupMinio(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	content := []byte("hello minio")
	reader := bytes.NewReader(content)
	objectName := "tests/" + uuid.New().String() + "/file.txt"

	opts := appminio.UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"x-amz-meta-test": "value",
		},
		CacheControl: "no-cache",
		StorageClass: "",
	}

	// Upload
	info, err := adapter.UploadStream(ctx, objectName, reader, int64(len(content)), opts)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, objectName, info.Key)
	require.Equal(t, int64(len(content)), info.Size)
	require.Equal(t, "text/plain", info.ContentType)

	// Exists
	exists, err := adapter.ObjectExists(ctx, objectName)
	require.NoError(t, err)
	require.True(t, exists)

	// Metadata
	meta, err := adapter.GetObjectMetadata(ctx, objectName)
	require.NoError(t, err)
	require.Equal(t, objectName, meta.Key)
	require.Equal(t, int64(len(content)), meta.Size)
	require.Equal(t, "text/plain", meta.ContentType)
}

func TestMinioAdapterDeleteAndNotExists(t *testing.T) {
	adapter, cleanup := setupMinio(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	content := []byte("to be deleted")
	reader := bytes.NewReader(content)
	objectName := "tests/" + uuid.New().String() + "/delete.txt"

	_, err := adapter.UploadStream(ctx, objectName, reader, int64(len(content)), appminio.UploadOptions{})
	require.NoError(t, err)

	// Убедимся, что есть
	exists, err := adapter.ObjectExists(ctx, objectName)
	require.NoError(t, err)
	require.True(t, exists)

	// Удаляем
	err = adapter.DeleteObject(ctx, objectName)
	require.NoError(t, err)

	// Теперь не должно существовать
	exists, err = adapter.ObjectExists(ctx, objectName)
	require.NoError(t, err)
	require.False(t, exists)

	// GetObjectMetadata должен вернуть CodeFileNotFound
	meta, err := adapter.GetObjectMetadata(ctx, objectName)
	require.Nil(t, meta)
	require.Error(t, err)

	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	require.Equal(t, apperrors.CodeFileNotFound, appErr.Code)
}

func TestMinioAdapterListObjectsAndListFiles(t *testing.T) {
	adapter, cleanup := setupMinio(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	basePrefix := "tests/" + uuid.New().String()

	files := []string{
		basePrefix + "/a/file1.txt",
		basePrefix + "/a/file2.txt",
		basePrefix + "/b/file3.txt",
	}

	for _, name := range files {
		_, err := adapter.UploadStream(ctx, name, bytes.NewReader([]byte("x")), 1, appminio.UploadOptions{})
		require.NoError(t, err)
	}

	// ListObjects с recursive=false по префиксу "basePrefix/a"
	objs, err := adapter.ListObjects(ctx, basePrefix+"/a", true)
	require.NoError(t, err)
	require.Len(t, objs, 2)

	// ListFiles (recursive=true по умолчанию) по basePrefix
	all, err := adapter.ListFiles(ctx, basePrefix)
	require.NoError(t, err)
	require.Len(t, all, 3)

	keys := make(map[string]struct{})
	for _, o := range all {
		keys[o.Key] = struct{}{}
	}
	for _, name := range files {
		_, ok := keys[name]
		require.True(t, ok, "expected key %s in ListFiles", name)
	}
}

func TestMinioAdapterGenerateStoragePath(t *testing.T) {
	adapter, cleanup := setupMinio(t)
	defer cleanup()

	taskID := uuid.New()
	filename := "doc.pdf"

	path := adapter.GenerateStoragePath(taskID, filename)

	require.Contains(t, path, adapter.BucketName())
	require.Contains(t, path, taskID.String())
	require.Contains(t, path, filename)
}
