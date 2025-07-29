package contract

type Store interface {
	MarkCompleted(key string)
	IsCompleted(key string) bool
}
