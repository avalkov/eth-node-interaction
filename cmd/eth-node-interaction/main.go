package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/avalkov/eth-node-interaction/internal/config"
	rpccodecs "github.com/avalkov/eth-node-interaction/internal/rpc_codecs"
	rpcservices "github.com/avalkov/eth-node-interaction/internal/rpc_services"
	dbstorage "github.com/avalkov/eth-node-interaction/internal/storage/db"
	txfetcher "github.com/avalkov/eth-node-interaction/internal/tx_fetcher"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/rpc"
	"github.com/xo/dburl"
)

func main() {
	if err := runService(); err != nil {
		log.Fatal(err)
	}
}

func runService() error {

	cfg, err := config.NewConfig(".env")
	if err != nil {
		return fmt.Errorf("creating config failed: %s", err)
	}

	client, err := ethclient.Dial(cfg.EthNodeUrl)
	if err != nil {
		return err
	}

	parsedConnectionUrl, err := dburl.Parse(cfg.DbConnectionUrl)
	if err != nil {
		return err
	}

	storage, err := dbstorage.NewStorage(parsedConnectionUrl.Driver, parsedConnectionUrl.DSN)
	if err != nil {
		return err
	}

	if err := storage.ExecuteMigrations(context.Background()); err != nil {
		return err
	}

	server := rpc.NewServer()

	codec := rpccodecs.NewTranslateMethodsCodec()
	server.RegisterCodec(codec, "application/json")
	server.RegisterCodec(codec, "application/json;charset=UTF-8")

	txFetcher := txfetcher.NewTxFetcher(storage, client)

	if err := server.RegisterService(rpcservices.NewLimeService(txFetcher, nil), ""); err != nil {
		return err
	}

	http.Handle("/", server)

	return http.ListenAndServe("localhost:1337", nil)
}
