module github.com/dogesuite/doged/btcec/v2

go 1.17

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1
	github.com/dogesuite/doged/chaincfg/chainhash v1.0.1
)

require github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect

replace github.com/dogesuite/doged/chaincfg/chainhash => ../chaincfg/chainhash
