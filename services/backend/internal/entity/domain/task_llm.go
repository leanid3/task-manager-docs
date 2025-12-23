package domain

// TODO! если треубется сохранить результат в базе, то нужно переопределить слой Adapter
type TaskLLM struct {
	Task
	StoragePath string      `json:"storage_path,omitempty"`
	StorageSize int64       `json:"storage_size,omitempty"`
	Metadata    LLMMetadata `json:"metadata,omitempty"`
}

const ContentTypeProtocol = "application/octet-stream"

type LLMMetadata struct {
	Filename    string `json:"filename,omitempty"`
	Filesize    int64  `json:"filesize,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}
