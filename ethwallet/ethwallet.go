package ethwallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/horizon-games/ethkit/ethcoder"
	"github.com/horizon-games/ethkit/ethrpc"
	"github.com/pkg/errors"
)

type Wallet struct {
	hdnode  *HDNode
	jsonrpc *ethrpc.JSONRPC
}

func NewWalletFromHDNode(hdnode *HDNode, path string) (*Wallet, error) {
	var err error
	var derivationPath accounts.DerivationPath

	if path == "" {
		derivationPath = DefaultBaseDerivationPath
	} else {
		derivationPath, err = ParseDerivationPath(path)
	}

	err = hdnode.DerivePath(derivationPath)
	if err != nil {
		return nil, err
	}

	return &Wallet{hdnode: hdnode}, nil
}

func NewWalletFromRandomEntropy(bitSize int, path string) (*Wallet, error) {
	hdnode, err := NewHDNodeFromRandomEntropy(bitSize, nil)
	if err != nil {
		return nil, err
	}
	return NewWalletFromHDNode(hdnode, path)
}

func NewWalletFromMnemonic(mnemonic string, path string) (*Wallet, error) {
	hdnode, err := NewHDNodeFromMnemonic(mnemonic, nil)
	if err != nil {
		return nil, err
	}
	return NewWalletFromHDNode(hdnode, path)
}

func (w *Wallet) Transactor() *bind.TransactOpts {
	return bind.NewKeyedTransactor(w.hdnode.PrivateKey())
}

func (w *Wallet) Provider() *ethrpc.JSONRPC {
	return w.jsonrpc
}

func (w *Wallet) SetProvider(provider *ethrpc.JSONRPC) error {
	w.jsonrpc = provider
	return nil
}

func (w *Wallet) DerivePath(path accounts.DerivationPath) (common.Address, error) {
	err := w.hdnode.DerivePath(path)
	if err != nil {
		return common.Address{}, err
	}
	return w.hdnode.Address(), nil
}

func (w *Wallet) DeriveAccountIndex(accountIndex uint32) error {
	return w.hdnode.DeriveAccountIndex(accountIndex)
}

func (w *Wallet) Accounts() []common.Address {
	// TODO
	return nil
}

func (w *Wallet) Address() common.Address {
	return w.hdnode.Address()
}

func (w *Wallet) PrivateKey() *ecdsa.PrivateKey {
	return w.hdnode.PrivateKey()
}

func (w *Wallet) PublicKey() *ecdsa.PublicKey {
	return w.hdnode.PublicKey()
}

func (w *Wallet) PrivateKeyHex() string {
	privateKeyBytes := crypto.FromECDSA(w.hdnode.PrivateKey())
	return hexutil.Encode(privateKeyBytes)[4:]
}

func (w *Wallet) PublicKeyHex() string {
	publicKeyBytes := crypto.FromECDSAPub(w.hdnode.PublicKey())
	return hexutil.Encode(publicKeyBytes)[4:]
}

func (w *Wallet) SignTx(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, w.hdnode.PrivateKey())
	if err != nil {
		return nil, err
	}

	msg, err := signedTx.AsMessage(types.HomesteadSigner{})
	if err != nil {
		return nil, err
	}

	sender := msg.From()
	if sender != w.hdnode.Address() {
		return nil, errors.Errorf("signer mismatch: expected %s, got %s", w.hdnode.Address().Hex(), sender.Hex())
	}

	return signedTx, nil
}

func (w *Wallet) SignMessage(msg []byte) ([]byte, error) {
	m := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	h := crypto.Keccak256([]byte(m))

	sig, err := crypto.Sign(h, w.hdnode.PrivateKey())
	if err != nil {
		return []byte{}, err
	}
	sig[64] += 27

	return sig, nil
}

// TODO ..
func (w *Wallet) SignTypedData(domainHash [32]byte, hashStruct [32]byte) ([]byte, error) {
	EIP191_HEADER := "0x1901000000000000000000000000000000000000000000000000000000000000"
	eip191Header, err := ethcoder.HexDecode(EIP191_HEADER)
	if err != nil {
		return []byte{}, err
	}

	preHash, err := ethcoder.SolidityPack([]string{"bytes", "bytes32"}, []interface{}{eip191Header, domainHash})
	if err != nil {
		return []byte{}, err
	}

	hashPack, err := ethcoder.SolidityPack([]string{"bytes", "bytes32"}, []interface{}{preHash, hashStruct})
	if err != nil {
		return []byte{}, err
	}
	hashBytes := crypto.Keccak256(hashPack)

	ethsigNoType, err := w.SignMessage(hashBytes)
	if err != nil {
		return []byte{}, err
	}
	ethsigNoType = append(ethsigNoType, 2) // because

	return ethsigNoType, nil
}

func (w *Wallet) GetBalance(ctx context.Context) (*big.Int, error) {
	balance, err := w.jsonrpc.BalanceAt(ctx, w.hdnode.Address(), nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (w *Wallet) GetTransactionCount(ctx context.Context) (uint64, error) {
	nonce, err := w.jsonrpc.PendingNonceAt(ctx, w.hdnode.Address())
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

// TODO
func (w *Wallet) SendTransaction() {
}
