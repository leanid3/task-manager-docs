## Распределение запроса между репликами task-manager и как публикуется сообщение

External Request → API Gateway 1 → Producer → tasks_llm (status=pending)
                           ↓
API Gateway 1 (consumer_llm, group="task_llm_group") → [skip pending]
API Gateway 2 (consumer_llm, group="task_llm_group") → [skip pending]  
API Gateway 3 (consumer_llm, group="task_llm_group") → [skip pending]

LLM Worker → API Gateway 1 → Producer → tasks_llm (status=processing)
                           ↓
API Gateway 2 (consumer_llm) → handleLLMProcessing()  // Kafka дала P1
