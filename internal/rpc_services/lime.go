package rpcservices

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/avalkov/eth-node-interaction/internal/model"
	"github.com/umbracle/fastrlp"
)

func NewLimeService(txFetcher txFetcher, authenticator authenticator) *Lime {
	return &Lime{
		txFetcher:     txFetcher,
		authenticator: authenticator,
	}
}

func (l *Lime) GetEthTransactions(r *http.Request, args *[]string, reply *GetEthTransactionsReply) error {
	txs := (*args)[0]

	var token *string
	if len((*args)) == 2 {
		token = &(*args)[1]

		if err := l.authenticator.VerifyToken(*token); err != nil {
			return err
		}
	}

	parser := &fastrlp.Parser{}
	txHashes, err := parser.Parse(unhex(txs))
	if err != nil {
		return err
	}

	count := txHashes.Elems()

	var wg sync.WaitGroup
	wg.Add(count)

	results := make(chan model.Transaction, count)

	for i := 0; i < count; i++ {
		value := txHashes.Get(i)
		hash, err := value.GetString()
		if err != nil {
			return err
		}

		go l.txFetcher.FetchTx(r.Context(), token, hash, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		reply.Transactions = append(reply.Transactions, res)
	}

	if len(reply.Transactions) != count {
		return fmt.Errorf("failed to fetch transactions")
	}

	return nil
}

func (l *Lime) GetAllTransactions(r *http.Request, _ *string, reply *GetEthTransactionsReply) error {
	transactions, err := l.txFetcher.FetchAllCachedTx(r.Context())
	if err != nil {
		return err
	}

	reply.Transactions = transactions

	return nil
}

func (l *Lime) GetMyTransactions(r *http.Request, args *[]string, reply *GetEthTransactionsReply) error {
	l.authenticator.VerifyToken((*args)[0])

	if len((*args)) == 1 {
		errors.New("missing token")
	}

	transactions, err := l.txFetcher.FetchAllCachedTxByToken(r.Context(), (*args)[1])
	if err != nil {
		return err
	}

	reply.Transactions = transactions

	return nil
}

func (l *Lime) Authenticate(r *http.Request, request *AuthenticateRequest, reply *AuthenticateReply) error {
	if request.Username == "" || request.Password == "" {
		return errors.New("invalid credentials")
	}

	tokenString, err := l.authenticator.Authenticate(r.Context(), request.Username, request.Password)
	if err != nil {
		return err
	}

	reply.Token = tokenString

	return nil
}

func unhex(str string) []byte {
	b, err := hex.DecodeString(strings.ReplaceAll(str, " ", ""))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

type AuthenticateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthenticateReply struct {
	Token string `json:"token"`
}

type GetEthTransactionsReply struct {
	Transactions []model.Transaction `json:"transactions"`
}

type txFetcher interface {
	FetchTx(ctx context.Context, token *string, hash string, results chan model.Transaction, wg *sync.WaitGroup)
	FetchAllCachedTx(ctx context.Context) ([]model.Transaction, error)
	FetchAllCachedTxByToken(ctx context.Context, token string) ([]model.Transaction, error)
}

type authenticator interface {
	Authenticate(ctx context.Context, username, password string) (string, error)
	VerifyToken(token string) error
}

type Lime struct {
	txFetcher     txFetcher
	authenticator authenticator
}
