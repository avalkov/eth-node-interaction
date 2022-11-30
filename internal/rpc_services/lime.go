package rpcservices

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/avalkov/eth-node-interaction/internal/model"
	"github.com/golang-jwt/jwt/v4"
	"github.com/umbracle/fastrlp"
)

func NewLimeService(txFetcher txFetcher, authenticator authenticator) *Lime {
	return &Lime{
		txFetcher:     txFetcher,
		authenticator: authenticator,
	}
}

func (l *Lime) GetEthTransactions(r *http.Request, args *interface{}, reply *GetEthTransactionsReply) error {
	fmt.Printf("GetEthTransactions: %+v\r\n", args)
	fmt.Printf("GetEthTransactions Kind: %v\r\n", reflect.TypeOf(*args).Kind())
	switch reflect.TypeOf(*args).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(*args)

		for i := 0; i < s.Len(); i++ {
			fmt.Println(s.Index(i))
		}
	}
	parser := &fastrlp.Parser{}
	txHashes, err := parser.Parse(unhex(""))
	if err != nil {
		fmt.Println(err)
		return err
	}

	count := txHashes.Elems()

	var wg sync.WaitGroup
	wg.Add(count)

	results := make(chan model.Transaction, count)

	for i := 0; i < count; i++ {
		value := txHashes.Get(0)
		hash, err := value.GetString()
		if err != nil {
			fmt.Println(err)
			return err
		}

		go l.txFetcher.FetchTx(r.Context(), nil, hash, results, &wg)
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

func (l *Lime) GetMyTransactions(r *http.Request, _ *string, reply *GetEthTransactionsReply) error {
	values := r.Header.Values(tokenHeader)
	if len(values) == 0 || values[0] == "" {
		return errors.New("token header not set")
	}

	return nil
}

func (l *Lime) Authenticate(r *http.Request, request *AuthenticateRequest, reply *AuthenticateReply) error {
	if request.Username == "" || request.Password == "" {
		return errors.New("invalid credentials")
	}

	if err := l.authenticator.Authenticate(request.Username, request.Password); err != nil {
		return err
	}

	expirationTime := time.Now().Add(666 * time.Minute)
	claims := &Claims{
		Username: request.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
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

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const tokenHeader = "Token"

var jwtKey = []byte("lime_secret_key")

type txFetcher interface {
	FetchTx(ctx context.Context, token *string, hash string, results chan model.Transaction, wg *sync.WaitGroup)
	FetchAllCachedTx(ctx context.Context) ([]model.Transaction, error)
}

type authenticator interface {
	Authenticate(username, password string) error
}

type Lime struct {
	txFetcher     txFetcher
	authenticator authenticator
}
