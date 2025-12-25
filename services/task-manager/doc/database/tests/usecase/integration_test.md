## Интеграционные тесты для постановки задач к llm

команда для запука 
```bash 
cd document-flow/services/backend
go test -tags=integration -timeout 5m -v -run ^TestTaskUC_CreateTask_Integration$ ./internal/usecase
```