# Как добавить новый тип задачи в task-manager

Эта инструкция описывает **универсальный шаблон** для добавления любого нового типа задачи в архитектуру `task-manager`. Она не привязана к конкретному способу взаимодействия (Kafka, HTTP, MinIO и т.д.) и может быть применена к любому сценарию.

---

## 🔑 1. Определите тип задачи (`TaskType`)

Добавьте новую константу в файл `internal/entity/domain/task_unified.go`:

```go
const (
    // ... существующие типы ...
    TaskTypeYourNewType TaskType = "your-new-type" // ← ваш уникальный lowercase-идентификатор
)
```

> 💡 Это значение будет использоваться для маршрутизации, регистрации обработчиков и сериализации.

---

## 🧱 2. Создайте структуру задачи

Создайте файл в `internal/entity/domain/`, например `task_your_new_type.go`:

```go
package domain

type TaskYourNewType struct {
    BaseTask // Обязательно embed-ьте BaseTask для реализации интерфейса Task
    // Ваши специфичные поля (с тегами json):
    Field1 string `json:"field1"`
    Field2 int    `json:"field2,omitempty"`
}
```

> ✅ Все задачи должны реализовывать интерфейс `Task` через embed `BaseTask`.  
> ✅ Используйте `json:` теги для корректной сериализации в Kafka/БД.

---

## 📦 3. Реализуйте usecase (бизнес-логика)

Создайте директорию `internal/usecase/your_new_type/` и файл `your_new_type_usecase.go`:

```go
package your_new_type

import (
    "app/internal/entity/domain"
    "app/internal/entity/repository"
    "context"
    "time"
)

type YourNewTypeUCInterface interface {
    CreateTask(ctx context.Context, input YourNewTypeInput) (domain.UUID, error)
    GetTaskByID(ctx context.Context, id domain.UUID) (domain.Task, error)
}

type YourNewTypeUC struct {
    taskRepo repository.Task
    // Другие зависимости (например, http.Client, minio.Client, broker.Producer и т.д.)
}

func NewYourNewTypeUC(taskRepo repository.Task, /* dependencies */) *YourNewTypeUC {
    return &YourNewTypeUC{taskRepo: taskRepo}
}

type YourNewTypeInput struct {
    Field1 string
    Field2 int
    RequestID string
}

func (uc *YourNewTypeUC) CreateTask(ctx context.Context, input YourNewTypeInput) (domain.UUID, error) {
    taskID := domain.NewUUID()
    traceID := domain.NewUUID()

    task := &domain.TaskYourNewType{
        BaseTask: domain.BaseTask{
            TaskID:    taskID,
            Status:    domain.TaskStatusPending,
            CreatedAt: time.Now(),
            TraceID:   &traceID,
            RequestID: input.RequestID,
            Metadata:  map[string]interface{}{"field1": input.Field1},
        },
        Field1: input.Field1,
        Field2: input.Field2,
    }

    // 1. Сохраните задачу в БД
    if err := uc.taskRepo.Create(ctx, task); err != nil {
        return domain.UUID{}, err
    }

    // 2. Выполните действия:
    //    - Для Kafka: uc.producer.Send(...)
    //    - Для HTTP: uc.httpClient.Post(...)
    //    - Для MinIO: uc.storageRepo.Upload(...)
    //    - Для синхронной обработки: выполните логику сразу

    // 3. Обновите статус задачи (если синхронно) или отправьте в очередь (если асинхронно)
    // uc.taskRepo.UpdateWithStatus(ctx, taskID, "", domain.TaskStatusProcessing)

    return taskID, nil
}
```

> 💡 Следуйте паттерну из `multiupload_usecase.go` или `task_llm_usecase.go`.  
> 💡 Обрабатывайте ошибки и обеспечивайте атомарность (например, при ошибке удаляйте созданные ресурсы).

---

## 🌐 4. Создайте HTTP-обработчик

Создайте директорию `internal/handlers/restapi/v1/your_new_type/` и файл `handler.go`:

```go
package your_new_type

import (
    "app/internal/entity/domain"
    "app/pkg/response"
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

type Handler struct {
    uc  YourNewTypeUCInterface
    cfg *Config
    l   Logger
}

func New(uc YourNewTypeUCInterface, cfg *Config, l Logger) *Handler {
    return &Handler{uc: uc, cfg: cfg, l: l}
}

// CreateYourNewTypeTask создает задачу вашего нового типа
// @Summary Создать задачу типа "your-new-type"
// @Description Создает задачу для выполнения вашего нового типа операции.
// @Tags your-new-type
// @Accept json
// @Produce json
// @Param input body YourNewTypeInput true "Входные данные"
// @Success 202 {object} CreateYourNewTypeTaskResponse "Задача успешно создана"
// @Failure 400 {object} response.ErrorResponse "Неверный формат запроса"
// @Failure 500 {object} response.ErrorResponse "Ошибка при создании задачи"
// @Router /api/v1/your-new-type [post]
func (h *Handler) CreateYourNewTypeTask(c *gin.Context) {
    var input YourNewTypeInput
    if err := c.ShouldBindJSON(&input); err != nil {
        h.handleError(c, err, "bind_json")
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.cfg.Server.Timeout)*time.Second)
    defer cancel()

    taskID, err := h.uc.CreateTask(ctx, input)
    if err != nil {
        h.handleError(c, err, "create_task")
        return
    }

    c.JSON(http.StatusAccepted, response.Success(CreateYourNewTypeTaskResponse{TaskID: taskID}, c.GetString("request_id")))
}
```

> ✅ Используйте Gin для обработки запросов.  
> ✅ Добавьте Swagger-комментарии для автоматической генерации документации.

---

## 🚦 5. Зарегистрируйте маршрут

В файле `internal/handlers/restapi/v1/router.go` добавьте новый маршрут:

```go
func SetupRoutes(r *gin.Engine, uc *usecase.Usecase, cfg *config.Config) {
    v1 := r.Group("/api/v1")
    {
        // ... другие группы ...
        yourNewTypeHandler := your_new_type.New(uc.YourNewTypeUC, cfg, logger)
        v1.POST("/your-new-type", yourNewTypeHandler.CreateYourNewTypeTask)
    }
}
```

---

## 🔄 6. (Опционально) Добавьте обработчик результата

Если задача асинхронная и использует Kafka:
1. Реализуйте `TaskProcessor` в `internal/usecase/your_new_type/processor.go`.
2. Зарегистрируйте его в `TaskProcessorFactory` через `Register(TaskTypeYourNewType, processor)`.
3. Создайте Kafka-консьюмер в `internal/handlers/broker/workflows/your_new_type.go`.

Если задача синхронная — этот шаг не требуется.

---

## 🧪 7. Напишите тесты

Добавьте unit-тесты для usecase и handler, следуя паттерну из существующих тестов (`*_test.go`).

---

> Эта инструкция универсальна и покрывает все случаи: от простых синхронных задач до сложных асинхронных с Kafka. Вы можете адаптировать каждый шаг под свои нужды.