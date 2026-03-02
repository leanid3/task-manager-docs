package kafka

// MessageInterface представляет абстракцию сообщения для отправки в Kafka
// Это позволяет изолировать pkg/kafka от доменных моделей
type MessageInterface interface {
	ToTopic() string
	ToKey() string
	ToHeaders() map[string]string
	ToValue() []byte
}

// BasicMessage реализация базового сообщения для тестов
type BasicMessage struct {
	topic   string
	key     string
	headers map[string]string
	value   []byte
}

func NewBasicMessage(topic, key string, headers map[string]string, value []byte) *BasicMessage {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &BasicMessage{
		topic:   topic,
		key:     key,
		headers: headers,
		value:   value,
	}
}

func (m *BasicMessage) ToTopic() string {
	return m.topic
}

func (m *BasicMessage) ToKey() string {
	return m.key
}

func (m *BasicMessage) ToHeaders() map[string]string {
	return m.headers
}

func (m *BasicMessage) ToValue() []byte {
	return m.value
}