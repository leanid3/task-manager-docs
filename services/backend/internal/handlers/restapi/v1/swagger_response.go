package v1

// @name SwaggerCreateTaskSuccess
type SwaggerCreateTaskSuccess struct {
	Data      CreateTaskResponse `json:"data"`
	RequestID string             `json:"request_id,omitempty"`
	Timestamp string             `json:"timestamp"`
}

// @name SwaggerGetTaskSuccess
type SwaggerGetTaskSuccess struct {
	Data      GetTaskResponse `json:"data"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp string          `json:"timestamp"`
}
