// internal/entity/domain/broker_command.go

package domain

// BrokerCommand — обобщенный тип для команд, отправляемых в брокер.
// Использует generics для обеспечения типобезопасности.
type BrokerCommand[T any] struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   T
}

// Для обратной совместимости можно использовать псевдоним.
// type TaskCommand = BrokerCommand[any]
