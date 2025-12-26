package minio

import (
	"io"
	"time"
)

// UploadOptions опции для загрузки файла
type UploadOptions struct {
	ContentType  string
	Metadata     map[string]string
	CacheControl string
	StorageClass string
}

// ObjectInfo информация об объекте в хранилище
type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
	Metadata     map[string]string
}

// DownloadOptions опции для скачивания файла
type DownloadOptions struct {
	VersionID string
}
type StreamReader interface {
	io.Reader
	io.Seeker
	io.Closer
}

// Config - конфигурация для MinIO коннектора
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Prefix    string
	UseSSL    bool
	Region    string
	Timeout   time.Duration
}
