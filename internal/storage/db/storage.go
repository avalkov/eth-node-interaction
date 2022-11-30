package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/avalkov/eth-node-interaction/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewStorage(driver, dsn string) (*storage, error) {
	if !strings.Contains(dsn, "sslmode") {
		dsn = fmt.Sprintf("%s sslmode=disable", dsn)
	}
	db, err := sqlx.Connect(driver, dsn)
	return &storage{db: db}, err
}

func (s *storage) ExecuteMigrations(ctx context.Context) error {
	return s.executeMigrations(ctx, s.db)
}

func (s *storage) GetTx(ctx context.Context, hash string) (model.Transaction, error) {
	var transactions []model.Transaction
	if err := s.db.SelectContext(ctx, &transactions, s.db.Rebind(`SELECT * FROM transaction WHERE transaction_hash = ?`), hash); err != nil {
		return model.Transaction{}, err
	}
	if len(transactions) == 0 {
		return model.Transaction{}, fmt.Errorf("tx (%s) not found.", hash)
	}
	return transactions[0], nil
}

func (s *storage) StoreTx(ctx context.Context, transaction model.Transaction, token *string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, s.db.Rebind(`INSERT INTO transaction (transaction_hash, transaction_status, block_hash, block_number,
    from_address, to_address, contract_address, logs_count, input, value) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`),
		transaction.TransactionHash, transaction.TransactionStatus, transaction.BlockHash, transaction.BlockNumber,
		transaction.From, transaction.To, transaction.ContractAddress,
		transaction.LogsCount, transaction.Input, transaction.Value); err != nil {
		return err
	}

	if token != nil {
		if _, err := tx.ExecContext(ctx, s.db.Rebind(`INSERT INTO token_transaction (token, transaction_hash) VALUES(?, ?) 
    ON CONFLICT DO NOTHING`), *token, transaction.TransactionHash); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *storage) GetAllTxs(ctx context.Context) ([]model.Transaction, error) {
	var transactions []model.Transaction
	if err := s.db.SelectContext(ctx, &transactions, `SELECT * FROM transaction`); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (s *storage) GetTxsByToken(ctx context.Context, token string) ([]model.Transaction, error) {
	var transactions []model.Transaction
	if err := s.db.SelectContext(ctx, &transactions, s.db.Rebind(`SELECT t.transaction_hash, t.transaction_status, 
    t.block_hash, t.block_number, t.from_address, t.to_address, t.contract_address, t.logs_count, t.input, 
    t.value FROM transaction AS t INNER JOIN token_transaction AS tt ON 
    t.transaction_hash = tt.transaction_hash WHERE tt.token = ?`), token); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (s *storage) IsUserExisting(ctx context.Context, username, password string) error {
	row := s.db.QueryRowContext(ctx, s.db.Rebind(`SELECT COUNT(*) FROM users WHERE username = ? AND password = ?`), username, password)
	errNotFound := errors.New("user not found")
	if row == nil {
		return errNotFound
	}

	var count int
	if err := row.Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		return errNotFound
	}

	return nil
}

type storage struct {
	db *sqlx.DB
}
