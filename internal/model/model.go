package model

type TxStatus int

const (
	Failed TxStatus = iota
	Successful
	Pending
)

type Transaction struct {
	TransactionHash   string   `json:"transactionHash" db:"transaction_hash"`
	TransactionStatus TxStatus `json:"transactionStatus" db:"transaction_status"`
	BlockHash         *string  `json:"blockHash" db:"block_hash"`
	BlockNumber       *uint64  `json:"blockNumber" db:"block_number"`
	From              string   `json:"from" db:"from_address"`
	To                *string  `json:"to" db:"to_address"`
	ContractAddress   *string  `json:"contractAddress" db:"contract_address"`
	LogsCount         *int     `json:"logsCount" db:"logs_count"`
	Input             string   `json:"input" db:"input"`
	Value             string   `json:"value" db:"value"`
}
