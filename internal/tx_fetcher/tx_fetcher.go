package txfetcher

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/avalkov/eth-node-interaction/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewTxFetcher(storage storage, client client) *txFetcher {
	return &txFetcher{
		storage: storage,
		client:  client,
	}
}

func (tf *txFetcher) FetchTx(ctx context.Context, token *string, txHashes []string) ([]model.Transaction, error) {
	count := len(txHashes)

	var wg sync.WaitGroup
	wg.Add(count)

	results := make(chan model.Transaction, count)

	for i := 0; i < count; i++ {
		go tf.fetchTx(ctx, token, txHashes[i], results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	transactions := []model.Transaction{}
	for res := range results {
		transactions = append(transactions, res)
	}

	if len(transactions) != count {
		return nil, fmt.Errorf("failed to fetch transactions: %v", txHashes)
	}

	return transactions, nil
}

func (tf *txFetcher) fetchTx(ctx context.Context, token *string, hash string, results chan model.Transaction, wg *sync.WaitGroup) {
	defer wg.Done()

	tx, err := tf.storage.GetTx(ctx, hash)
	if err != nil {
		// The storage layer returns error if not found
		log.Println(err)
	}

	txHash := common.HexToHash(hash)

	rawTx, isPending, err := tf.client.TransactionByHash(ctx, txHash)
	if err != nil {
		log.Println(err)
		return
	}

	var receipt *types.Receipt

	if !isPending {
		receipt, err = tf.client.TransactionReceipt(ctx, txHash)
		if err != nil {
			log.Println(err)
			return
		}
	}

	tx, err = parseRawTx(rawTx, receipt, isPending)
	if err != nil {
		log.Println(err)
		return
	}

	if !isPending {
		go func() {
			ctxWithTimeout, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelFunc()
			if err := tf.storage.StoreTx(ctxWithTimeout, tx, token); err != nil {
				log.Println(fmt.Errorf("failed to store tx (%s): %s", tx.TransactionHash, err))
			}
		}()
	}

	results <- tx
}

func (tf *txFetcher) FetchAllCachedTx(ctx context.Context) ([]model.Transaction, error) {
	return tf.storage.GetAllTxs(ctx)
}

func (tf *txFetcher) FetchAllCachedTxByToken(ctx context.Context, token string) ([]model.Transaction, error) {
	return tf.storage.GetTxsByToken(ctx, token)
}

func parseRawTx(tx *types.Transaction, receipt *types.Receipt, isPending bool) (model.Transaction, error) {
	var status model.TxStatus

	if isPending {
		status = model.Pending
	} else {
		status = model.TxStatus(receipt.Status)
	}

	txMsg := getTransactionMessage(tx)

	parsedTx := model.Transaction{
		TransactionHash:   tx.Hash().Hex(),
		TransactionStatus: status,
		From:              txMsg.From().Hex(),
		Input:             hex.EncodeToString(tx.Data()),
		Value:             tx.Value().String(),
	}

	if tx.To() != nil {
		to := tx.To().Hex()
		parsedTx.To = &to
	}

	if receipt != nil {
		if tx.To() == nil {
			contractAddress := receipt.ContractAddress.Hex()
			if contractAddress != "" {
				parsedTx.ContractAddress = &contractAddress
			}
		}

		blockHash := receipt.BlockHash.Hex()
		parsedTx.BlockHash = &blockHash

		blockNumber := receipt.BlockNumber.Uint64()
		parsedTx.BlockNumber = &blockNumber

		logsCount := len(receipt.Logs)
		parsedTx.LogsCount = &logsCount
	}

	return parsedTx, nil
}

func getTransactionMessage(tx *types.Transaction) types.Message {
	msg, err := tx.AsMessage(types.LatestSignerForChainID(tx.ChainId()), nil)
	if err != nil {
		log.Println(err)
	}
	return msg
}

type storage interface {
	GetTx(ctx context.Context, hash string) (model.Transaction, error)
	StoreTx(ctx context.Context, transaction model.Transaction, token *string) error
	GetAllTxs(ctx context.Context) ([]model.Transaction, error)
	GetTxsByToken(ctx context.Context, token string) ([]model.Transaction, error)
}

type client interface {
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type txFetcher struct {
	storage storage
	client  client
}
