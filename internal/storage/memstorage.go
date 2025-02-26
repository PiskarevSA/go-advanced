package storage

type MemStorage struct{}

func NewMemStorage() *MemStorage {
	return &MemStorage{}
}
