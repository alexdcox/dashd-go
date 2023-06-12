module github.com/dashpay/dashd-go/btcec/v2

go 1.18

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
)

require github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect

replace (
	github.com/dashpay/dashd-go => ../
	github.com/dashpay/dashd-go/addrmgr => ./addrmgr
	github.com/dashpay/dashd-go/blockchain => ./blockchain
	github.com/dashpay/dashd-go/btcec => ./btcec
	github.com/dashpay/dashd-go/btcjson => ./btcjson
	github.com/dashpay/dashd-go/btcutil => ./btcutil
	github.com/dashpay/dashd-go/chaincfg => ./chaincfg
	github.com/dashpay/dashd-go/cmd => ./cmd
	github.com/dashpay/dashd-go/connmgr => ./connmgr
	github.com/dashpay/dashd-go/database => ./database
	github.com/dashpay/dashd-go/docs => ./docs
	github.com/dashpay/dashd-go/integration => ./integration
	github.com/dashpay/dashd-go/limits => ./limits
	github.com/dashpay/dashd-go/mempool => ./mempool
	github.com/dashpay/dashd-go/mining => ./mining
	github.com/dashpay/dashd-go/netsync => ./netsync
	github.com/dashpay/dashd-go/peer => ./peer
	github.com/dashpay/dashd-go/release => ./release
	github.com/dashpay/dashd-go/rpcclient => ./rpcclient
	github.com/dashpay/dashd-go/txscript => ./txscript
	github.com/dashpay/dashd-go/wire => ./wire
)