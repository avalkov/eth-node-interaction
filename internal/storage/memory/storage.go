package memory

import (
	"context"
	"fmt"

	"github.com/avalkov/eth-node-interaction/internal/model"
)

func NewStorage() *storage {
	return &storage{
		store: make(map[string]model.Transaction),
	}
}

func (s *storage) GetTx(ctx context.Context, hash string) (model.Transaction, error) {
	if tx, ok := s.store[hash]; ok {
		return tx, nil
	}
	return model.Transaction{}, fmt.Errorf("tx (%s) not found in storage", hash)
}

func (s *storage) StoreTx(ctx context.Context, transaction model.Transaction) error {
	s.store[transaction.TransactionHash] = transaction
	return nil
}

type storage struct {
	store map[string]model.Transaction
}
