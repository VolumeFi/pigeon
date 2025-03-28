package evm

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/VolumeFi/whoops"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum"
	etherum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	etherumtypes "github.com/ethereum/go-ethereum/core/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/palomachain/paloma/v2/x/evm/types"
	skywaytypes "github.com/palomachain/paloma/v2/x/skyway/types"
	valset "github.com/palomachain/paloma/v2/x/valset/types"
	compassABI "github.com/palomachain/pigeon/chain/evm/abi/compass"
	feemgrABI "github.com/palomachain/pigeon/chain/evm/abi/feemgr"
	"github.com/palomachain/pigeon/chain/paloma"
	"github.com/palomachain/pigeon/config"
	"github.com/palomachain/pigeon/errors"
	"github.com/palomachain/pigeon/internal/liblog"
	"github.com/palomachain/pigeon/util/slice"
	arbcommon "github.com/roodeag/arbitrum/common"
	arbclient "github.com/roodeag/arbitrum/ethclient"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type StoredContract struct {
	ABI    abi.ABI
	Source []byte
}

/*
Do not delete hello.json contract. It's used for tests!
*/
var (
	//go:embed contracts/*.json
	contractsFS embed.FS

	readOnce   sync.Once
	_contracts = make(map[string]StoredContract)
)

func StoredContracts() map[string]StoredContract {
	readOnce.Do(func() {
		err := fs.WalkDir(contractsFS, ".", func(path string, d fs.DirEntry, err error) error {
			logger := log.WithFields(log.Fields{
				"path": path,
			})
			if d.IsDir() {
				return nil
			}
			file, err := contractsFS.Open(path)
			if err != nil {
				logger.WithFields(log.Fields{
					"err": err,
				}).Fatal("couldn't open contract file")
			}

			contractName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

			// we need to store body locally, so reading it here first and
			// using bytes.NewBuffer few lines down.
			body := whoops.Must(io.ReadAll(file))

			evmabi, err := abi.JSON(bytes.NewBuffer(body))
			if err != nil {
				logger.WithFields(log.Fields{
					"err": err,
				}).Fatal("couldn't read contract file")
			}

			_contracts[contractName] = StoredContract{
				ABI:    evmabi,
				Source: body,
			}
			return nil
		})
		if err != nil {
			log.WithField("err", err).Error("error iterating over the stored contracts")
		}
	})
	return _contracts
}

//go:generate mockery --name=PalomaClienter
type PalomaClienter interface {
	NewStatus() paloma.StatusUpdater
	AddMessageEvidence(ctx context.Context, queueTypeName string, messageID uint64, proof proto.Message) error
	SetPublicAccessData(ctx context.Context, queueTypeName string, messageID uint64, valsetID uint64, data []byte) error
	SetErrorData(ctx context.Context, queueTypeName string, messageID uint64, data []byte) error
	QueryGetEVMValsetByID(ctx context.Context, id uint64, chainID string) (*types.Valset, error)
	SendBatchSendToEVMClaim(ctx context.Context, claim skywaytypes.MsgBatchSendToRemoteClaim) error
	SendSendToPalomaClaim(ctx context.Context, claim skywaytypes.MsgSendToPalomaClaim) error
	SendLightNodeSaleClaim(ctx context.Context, claim skywaytypes.MsgLightNodeSaleClaim) error
	QueryLastObservedSkywayNonceByAddr(ctx context.Context, chainReferenceID string, orchestrator string) (uint64, error)
	QueryBatchRequestByNonce(ctx context.Context, nonce uint64, contract string) (skywaytypes.OutgoingTxBatch, error)
	QueryGetLatestPublishedSnapshot(ctx context.Context, chainReferenceID string) (*valset.Snapshot, error)
	QueryGetSnapshotByID(ctx context.Context, id uint64) (*valset.Snapshot, error)
	QueryUnobservedBlocksByValidator(ctx context.Context, chainReferenceID string, orchestrator string) ([]uint64, error)
}

type Client struct {
	config config.EVM

	addr     ethcommon.Address
	keystore *keystore.KeyStore

	conn   ethClientConn
	arbcon *arbclient.Client

	paloma    PalomaClienter
	mevClient mevClient
}

func (c Client) GetEthClient() ethClientConn {
	return c.conn
}

var _ ethClientConn = &ethclient.Client{}

//go:generate mockery --name=mevClient --inpackage --testonly
type mevClient interface {
	Relay(context.Context, *big.Int, *ethtypes.Transaction) (common.Hash, error)
}

//go:generate mockery --name=ethClientConn --inpackage --testonly
type ethClientConn interface {
	ethClienter
	TransactionByHash(ctx context.Context, hash common.Hash) (tx *etherumtypes.Transaction, isPending bool, err error)
	TransactionReceipt(ctx context.Context, hash common.Hash) (*etherumtypes.Receipt, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*etherumtypes.Header, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*etherumtypes.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

type CompassBindingCaller interface {
	LastCheckpoint(opts *bind.CallOpts) ([32]byte, error)
	LastValsetId(opts *bind.CallOpts) (*big.Int, error)
	MessageIdUsed(opts *bind.CallOpts, arg0 *big.Int) (bool, error)
	CompassId(opts *bind.CallOpts) ([32]byte, error)
}

type CompassBindingTransactor interface {
	SubmitLogicCall(opts *bind.TransactOpts, consensus compassABI.Struct2, args compassABI.Struct4, fee_args compassABI.Struct5, messageId *big.Int, deadline *big.Int, relayer common.Address) (*ethtypes.Transaction, error)
	UpdateValset(opts *bind.TransactOpts, consensus compassABI.Struct2, newValset compassABI.Struct0, relayer common.Address, gas_estimate *big.Int) (*ethtypes.Transaction, error)
	CompassUpdateBatch(opts *bind.TransactOpts, consensus compassABI.Struct2, update_compass_args []compassABI.Struct4, deadline *big.Int, gas_estimate *big.Int, relayer common.Address) (*ethtypes.Transaction, error)
}

type CompassBindingFilterer interface {
	FilterLogicCallEvent(opts *bind.FilterOpts) (*compassABI.CompassLogicCallEventIterator, error)
	FilterValsetUpdated(opts *bind.FilterOpts) (*compassABI.CompassValsetUpdatedIterator, error)
	ParseLogicCallEvent(log ethtypes.Log) (*compassABI.CompassLogicCallEvent, error)
	ParseValsetUpdated(log ethtypes.Log) (*compassABI.CompassValsetUpdated, error)
	WatchLogicCallEvent(opts *bind.WatchOpts, sink chan<- *compassABI.CompassLogicCallEvent) (event.Subscription, error)
	WatchValsetUpdated(opts *bind.WatchOpts, sink chan<- *compassABI.CompassValsetUpdated) (event.Subscription, error)
}

//go:generate mockery --name=ethClienter --inpackage --testonly
type ethClienter interface {
	bind.ContractBackend
}

//go:generate mockery --name=CompassBinding
type CompassBinding interface {
	CompassBindingCaller
	CompassBindingTransactor
	CompassBindingFilterer
}

func (c *Client) init() error {
	return whoops.Try(func() {
		if !ethcommon.IsHexAddress(c.config.SigningKey) {
			whoops.Assert(errors.Unrecoverable(ErrInvalidAddress.Format(c.config.SigningKey)))
		}
		c.addr = ethcommon.HexToAddress(c.config.SigningKey)

		if c.keystore == nil {
			c.keystore = keystore.NewKeyStore(c.config.KeyringDirectory.Path(), keystore.StandardScryptN, keystore.StandardScryptP)
		}

		if !c.keystore.HasAddress(c.addr) {
			whoops.Assert(errors.Unrecoverable(ErrAddressNotFoundInKeyStore.Format(c.config.SigningKey, c.config.KeyringDirectory.Path())))
		}
		acc := accounts.Account{Address: c.addr}

		whoops.Assert(c.keystore.Unlock(acc, config.KeyringPassword(c.config.KeyringPassEnvName)))

		c.conn = whoops.Must(ethclient.Dial(c.config.BaseRPCURL))
	})
}

func (c *Client) injectArbClient() error {
	ac, err := arbclient.Dial(c.config.BaseRPCURL)
	if err != nil {
		return err
	}

	c.arbcon = ac
	return nil
}

func (c *Client) isArbitrumClient() bool {
	return c.arbcon != nil
}

func (c *Client) newCompass(addr common.Address) (CompassBinding, error) {
	return compassABI.NewCompass(addr, c.conn)
}

type executeSmartContractIn struct {
	ethClient ethClienter
	mevClient mevClient

	chainID       *big.Int
	gasAdjustment float64
	txType        uint8

	abi      abi.ABI
	contract common.Address

	signingAddr common.Address
	keystore    *keystore.KeyStore

	method    string
	arguments []any
	opts      callOptions

	gasEstimate *big.Int
}

func callSmartContract(
	ctx context.Context,
	args executeSmartContractIn,
) (*etherumtypes.Transaction, error) {
	logger := liblog.WithContext(ctx).WithFields(log.Fields{
		"chain-id":        args.chainID,
		"contract-addr":   args.contract,
		"method":          args.method,
		"arguments":       args.arguments,
		"gas-adjustments": args.gasAdjustment,
		"signing-addr":    args.signingAddr,
	})
	return whoops.TryVal(func() *etherumtypes.Transaction {
		packedBytes, err := args.abi.Pack(
			args.method,
			args.arguments...,
		)
		if err != nil {
			logger.
				WithField("error", err).
				Error("callSmartContract: error packing input")
		}
		whoops.Assert(err)

		nonce, err := args.ethClient.PendingNonceAt(ctx, args.signingAddr)
		if err != nil {
			logger.
				WithField("error", err).
				Error("callSmartContract: error calculating pending nonce")
		}
		whoops.Assert(err)

		gasPrice, err := args.ethClient.SuggestGasPrice(ctx)
		if err != nil {
			logger.
				WithField("error", err).
				Error("callSmartContract: error calculating suggested gas price")
		}
		whoops.Assert(err)

		// adjusting the gas price
		if args.txType != 2 && args.gasAdjustment > 1.0 {
			gasAdj := big.NewFloat(args.gasAdjustment)
			gasAdj = gasAdj.Mul(gasAdj, new(big.Float).SetInt(gasPrice))
			gasPrice, _ = gasAdj.Int(big.NewInt(0))
		}

		var gasTipCap *big.Int

		if args.txType == 2 {
			gasPrice = gasPrice.Mul(gasPrice, big.NewInt(2)) // double gas price for EIP-1559 tx
			gasTipCap, err = args.ethClient.SuggestGasTipCap(ctx)
			if err != nil {
				logger.
					WithField("error", err).
					Error("callSmartContract: error calling SuggestGasTipCap")
			}
			whoops.Assert(err)
			gasPrice = gasPrice.Add(gasPrice, gasTipCap)
			logger.WithFields(log.Fields{
				"gas-max-price": gasPrice,
				"gas-max-tip":   gasTipCap,
			}).Debug("adjusted eip-1559 gas price")
		} else {
			logger.WithFields(log.Fields{
				"gas-price": gasPrice,
			}).Debug("adjusted legacy gas price")
		}

		boundContract := bind.NewBoundContract(
			args.contract,
			args.abi,
			args.ethClient,
			args.ethClient,
			args.ethClient,
		)

		txOpts, err := bind.NewKeyStoreTransactorWithChainID(
			args.keystore,
			accounts.Account{Address: args.signingAddr},
			args.chainID,
		)
		if err != nil {
			logger.
				WithField("error", err).
				Error("callSmartContract: error calling bind.NewKeyStoreTransactorWithChainID")
		}
		whoops.Assert(err)

		txOpts.Nonce = big.NewInt(int64(nonce))
		txOpts.From = args.signingAddr

		value := new(big.Int)
		gasFeeCap := new(big.Int)
		var gasLimit uint64
		if args.txType == 2 {
			gasFeeCap = gasPrice
			gasLimit, err = estimateGasLimit(ctx, args.ethClient, txOpts, &args.contract, packedBytes, nil, gasTipCap, gasFeeCap, value)
			whoops.Assert(err)
		} else {
			gasLimit, err = estimateGasLimit(ctx, args.ethClient, txOpts, &args.contract, packedBytes, gasPrice, nil, nil, value)
			whoops.Assert(err)
		}
		logger.WithFields(log.Fields{
			"gas-limit": gasLimit,
		}).Debug("estimated gas limit")

		// In case we only want to estimate, now is the time to return.
		if args.opts.estimateOnly {
			return ethtypes.NewTx(
				&ethtypes.LegacyTx{
					Gas: gasLimit,
				})
		}

		// Once estimation is finished, we adjust the gas limit
		// to be sure that the tx will be included in the next block.
		txOpts.GasLimit = uint64(float64(gasLimit) * 1.5)
		if args.gasEstimate != nil && args.gasEstimate.Uint64() > txOpts.GasLimit {
			// If we have a gas estimate from pigeons, and it is greater than
			// what we just got, we use it
			txOpts.GasLimit = args.gasEstimate.Uint64()
		}

		// Add an additional buffer to cover the cost of gas verification
		// on Compass.
		// See https://github.com/VolumeFi/paloma/issues/2299
		txOpts.GasLimit = txOpts.GasLimit + 100_000

		if args.txType == 2 {
			txOpts.GasFeeCap = gasPrice
			txOpts.GasTipCap = gasTipCap
			txOpts.GasPrice = nil
			logger.WithFields(log.Fields{
				"gas-limit":     txOpts.GasLimit,
				"gas-max-price": txOpts.GasFeeCap,
				"gas-max-tip":   txOpts.GasTipCap,
				"nonce":         txOpts.Nonce,
				"from":          txOpts.From,
			}).Debug("executing eip-1559 tx")
		} else {
			txOpts.GasPrice = gasPrice
			txOpts.GasFeeCap = nil
			txOpts.GasTipCap = nil
			logger.WithFields(log.Fields{
				"gas-limit": txOpts.GasLimit,
				"gas-price": txOpts.GasPrice,
				"nonce":     txOpts.Nonce,
				"from":      txOpts.From,
			}).Debug("executing legacy tx")
		}

		// In case we want to relay, don't actually send the constructed TX
		if args.opts.useMevRelay && args.mevClient != nil {
			logger.Info("MEV Client set - setting TX to not execute")
			txOpts.NoSend = true
		}
		tx, err := boundContract.RawTransact(txOpts, packedBytes)
		if err != nil {
			logger.
				WithField("error", err).
				Error("callSmartContract: error calling boundContract.RawTransact")
		}
		whoops.Assert(err)

		if args.mevClient != nil {
			hash, err := args.mevClient.Relay(ctx, args.chainID, tx)
			logger.WithField("relay-hash", hash).Info("Attempted to MEV relay")
			if err != nil || hash != tx.Hash() {
				if err == nil {
					err = fmt.Errorf("hash mismatch, expected %s, got %s", tx.Hash(), hash)
				}
				logger.WithField("error", err).Error("callSmartContract: error calling mevClient.Relay")
				whoops.Assert(err)
			}
		}

		msg := "executed"
		logger.WithField("txOps-nosend", txOpts.NoSend).Info("Checking for no send")
		if txOpts.NoSend {
			msg = "relayed"
		}
		if args.txType == 2 {
			logger.WithFields(log.Fields{
				"tx-hash":          tx.Hash(),
				"tx-gas-limit":     tx.Gas(),
				"tx-gas-max-price": tx.GasFeeCap(),
				"tx-gas-max-tip":   tx.GasTipCap(),
				"tx-cost":          tx.Cost(),
			}).Debugf("eip-1559 tx %s", msg)
		} else {
			logger.WithFields(log.Fields{
				"tx-hash":      tx.Hash(),
				"tx-gas-limit": tx.Gas(),
				"tx-gas-price": tx.GasPrice(),
				"tx-cost":      tx.Cost(),
			}).Debugf("legacy tx %s", msg)
		}

		return tx
	})
}

// Copied from https://github.com/ethereum/go-ethereum/blob/e5d5e09faae48dac3723634e2b1813e4f2e89535/accounts/abi/bind/base.go#L357
// since the original implementation is not exported, but we need access to the gas limit to multiply it
// before we send the TX.
// More detail: https://github.com/VolumeFi/paloma/issues/1048
// And: https://github.com/VolumeFi/paloma/issues/1158#issuecomment-1896488916
func estimateGasLimit(ctx context.Context, c ethClienter, opts *bind.TransactOpts, contract *ethcommon.Address, input []byte, gasPrice, gasTipCap, gasFeeCap, value *big.Int) (uint64, error) {
	if contract != nil {
		// Gas estimation cannot succeed without code for method invocations.
		if code, err := c.PendingCodeAt(ctx, *contract); err != nil {
			return 0, err
		} else if len(code) == 0 {
			return 0, bind.ErrNoCode
		}
	}

	msg := ethereum.CallMsg{
		From:      opts.From,
		To:        contract,
		GasPrice:  gasPrice,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Value:     value,
		Data:      input,
	}

	return c.EstimateGas(ctx, msg)
}

func (c *Client) sign(ctx context.Context, bytes []byte) ([]byte, error) {
	return c.keystore.SignHash(
		accounts.Account{Address: c.addr},
		bytes,
	)
}

// FilterLogs will gather all logs given a FilterQuery. If it encounters an
// error saying that there are too many results in the provided block window,
// then it's going to try to do this using a "binary search" approach while
// splitting the  possible set in two, recursively.
func (c *Client) FilterLogs(ctx context.Context, fq etherum.FilterQuery, currBlockHeight *big.Int, fn func(logs []ethtypes.Log) bool) (bool, error) {
	found, err := filterLogs(ctx, c.conn, fq, currBlockHeight, true, fn)
	if err != nil {
		log.WithError(err).Error("error filtering logs")
	}

	return found, err
}

func (c *Client) TransactionByHash(ctx context.Context, txHash common.Hash) (*ethtypes.Transaction, bool, error) {
	return c.conn.TransactionByHash(ctx, txHash)
}

func (c *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethtypes.Receipt, error) {
	return c.conn.TransactionReceipt(ctx, txHash)
}

func (c *Client) BlockByHash(ctx context.Context, blockHash common.Hash) (*ethtypes.Block, error) {
	if c.isArbitrumClient() {
		return c.wrapArbitrumBlockByHash(ctx, blockHash)
	}
	return c.conn.BlockByHash(ctx, blockHash)
}

func (c *Client) wrapArbitrumBlockByHash(ctx context.Context, blockHash common.Hash) (*ethtypes.Block, error) {
	b, err := c.arbcon.BlockByHash(ctx, arbcommon.BytesToHash(blockHash.Bytes()))
	if err != nil {
		return nil, err
	}

	hdr := &ethtypes.Header{
		ParentHash:      ethcommon.Hash(b.Header().ParentHash),
		UncleHash:       ethcommon.Hash(b.Header().UncleHash),
		Coinbase:        ethcommon.Address(b.Header().Coinbase),
		Root:            ethcommon.Hash(b.Header().Root),
		TxHash:          ethcommon.Hash(b.Header().TxHash),
		ReceiptHash:     ethcommon.Hash(b.Header().ReceiptHash),
		Bloom:           ethtypes.Bloom(b.Header().Bloom),
		Difficulty:      b.Header().Difficulty,
		Number:          b.Header().Number,
		GasLimit:        b.Header().GasLimit,
		GasUsed:         b.Header().GasUsed,
		Time:            b.Header().Time,
		Extra:           b.Header().Extra,
		MixDigest:       ethcommon.Hash(b.Header().MixDigest),
		Nonce:           ethtypes.BlockNonce(b.Header().Nonce),
		BaseFee:         b.Header().BaseFee,
		WithdrawalsHash: (*ethcommon.Hash)(b.Header().WithdrawalsHash),
		ExcessBlobGas:   nil,
	}
	return ethtypes.NewBlockWithHeader(hdr), nil
}

//go:generate mockery --name=ethClientToFilterLogs --inpackage --testonly
type ethClientToFilterLogs interface {
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]etherumtypes.Log, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*etherumtypes.Header, error)
}

func shouldDoBinarySearchFromError(err error) bool {
	switch {
	case strings.Contains(err.Error(), "query returned more than 10000 results"):
		return true
	case strings.Contains(err.Error(), "eth_getLogs and eth_newFilter are limited to a 10,000 blocks range"):
		return true
	case strings.Contains(err.Error(), "block range is too wide"):
		return true
	case strings.Contains(err.Error(), "exceed maximum block range"):
		return true
	}

	return false
}

// filterLogs filters for logs in a recursive manner. If the server returns
// that the block range is too high, then it does a binary search for left and
// right sectin.
func filterLogs(
	ctx context.Context,
	ethClient ethClientToFilterLogs,
	fq etherum.FilterQuery,
	currBlockHeight *big.Int,
	// reverseOrder if set to true then it searches latest logs first
	reverseOrder bool,
	fn func(logs []ethtypes.Log) bool,
) (bool, error) {
	log.
		WithField("current-block-height", currBlockHeight).
		WithField("filter-query", fq).
		Trace("filtering logs")

	if currBlockHeight == nil {
		header, err := ethClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return false, err
		}
		currBlockHeight = header.Number
	}

	if fq.BlockHash == nil {
		if fq.ToBlock == nil {
			fq.ToBlock = currBlockHeight
		}
		if fq.FromBlock == nil {
			fq.FromBlock = big.NewInt(0)
		}
	}

	logs, err := ethClient.FilterLogs(ctx, fq)

	switch {
	case err == nil:
		// awesome!
		if len(logs) == 0 {
			return false, nil
		}
		slice.ReverseInplace(logs)
		return fn(logs), nil
	case shouldDoBinarySearchFromError(err):
		// this appears to be ropsten specifict, but keepeing the logic here just in case
		mid := big.NewInt(0).Sub(
			fq.ToBlock,
			fq.FromBlock,
		)
		mid.Div(mid, big.NewInt(2))
		mid.Add(fq.FromBlock, mid)

		left := func() (bool, error) {
			fqLeft := fq
			fqLeft.ToBlock = mid
			return filterLogs(
				ctx,
				ethClient,
				fqLeft,
				currBlockHeight,
				reverseOrder,
				fn,
			)
		}

		right := func() (bool, error) {
			fqRight := fq
			fqRight.FromBlock = big.NewInt(0).Add(mid, big.NewInt(1))
			return filterLogs(
				ctx,
				ethClient,
				fqRight,
				currBlockHeight,
				reverseOrder,
				fn,
			)
		}

		var first, second func() (bool, error)

		if reverseOrder {
			first, second = right, left
		} else {
			first, second = left, right
		}

		foundFirst, err := first()
		if err != nil {
			return false, err
		}

		if foundFirst {
			return true, nil
		}

		return second()

	}

	return false, err
}

func (c *Client) ExecuteSmartContract(
	ctx context.Context,
	chainID *big.Int,
	contractAbi abi.ABI,
	addr common.Address,
	opts callOptions,
	method string,
	arguments []any,
	gasEstimate *big.Int,
) (*etherumtypes.Transaction, error) {
	var mevClient mevClient = nil
	if opts.useMevRelay {
		mevClient = c.mevClient
		logrus.WithContext(ctx).WithField("mevClient", mevClient).WithField("c.mevClient", c.mevClient).Info("Using MEV relay")
	}

	return callSmartContract(
		ctx,
		executeSmartContractIn{
			ethClient:     c.conn,
			mevClient:     mevClient,
			chainID:       chainID,
			gasAdjustment: c.config.GasAdjustment,
			txType:        c.config.TxType,
			abi:           contractAbi,
			contract:      addr,
			signingAddr:   c.addr,
			keystore:      c.keystore,
			opts:          opts,

			method:      method,
			arguments:   arguments,
			gasEstimate: gasEstimate,
		},
	)
}

func (c *Client) BalanceAt(ctx context.Context, address common.Address, blockHeight uint64) (*big.Int, error) {
	var bh *big.Int
	if blockHeight > 0 {
		bh = new(big.Int).SetUint64(blockHeight)
	}
	return c.conn.BalanceAt(ctx, address, bh)
}

func (c *Client) FindBlockNearestToTime(ctx context.Context, startingHeight uint64, when time.Time) (uint64, error) {
	isTimeSetBeforeBlock := func(height uint64) (bool, error) {
		h, err := c.conn.HeaderByNumber(ctx, new(big.Int).SetUint64(height))
		if err != nil {
			return false, err
		}
		return h.Time < uint64(when.UTC().Unix()), nil
	}

	before, err := isTimeSetBeforeBlock(startingHeight)
	if err != nil {
		return 0, err
	}
	if !before {
		return 0, ErrStartingBlockIsInTheFuture
	}

	currBlockHeight, err := c.conn.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	from, to := startingHeight, currBlockHeight
	var res uint64
	for from <= to {
		err := whoops.Try(func() {
			mid := uint64(math.Round(float64(from+to) / 2))
			before := whoops.Must(isTimeSetBeforeBlock(mid))
			if before {
				res = mid
				from = mid + 1
			} else {
				to = mid - 1
			}
		})
		if err != nil {
			return 0, err
		}
	}

	if res == currBlockHeight {
		// in case the block has just been gerated, return the
		// previous block number.
		return res - 1, nil
	}

	return res, nil
}

func (c *Client) FindCurrentBlockNumber(ctx context.Context) (*big.Int, error) {
	header, err := c.conn.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	return header.Number, nil
}

func (c *Client) LastValsetID(ctx context.Context, addr common.Address) (*big.Int, error) {
	logger := liblog.WithContext(ctx).WithField("address", addr.String())
	logger.Debug("called LastValsetID in EVM client")

	cmps, err := c.newCompass(addr)
	if err != nil {
		logger.WithError(err).Error("LastValsetID: error instantiating compass")
		return nil, fmt.Errorf("error instantiating compass binding: %w", err)
	}

	callOpts := &bind.CallOpts{
		Pending: false,
		Context: ctx,
	}
	return cmps.LastValsetId(callOpts)
}

func (c *Client) QueryUserFunds(ctx context.Context, feemgraddr common.Address, palomaAddress [32]byte) (*big.Int, error) {
	logger := liblog.WithContext(ctx).WithField("address", feemgraddr.String())
	logger.Debug("called QueryUserFunds in EVM client")

	fm, err := feemgrABI.NewFeemgr(feemgraddr, c.conn)
	if err != nil {
		logger.WithError(err).Error("QueryUserFunds: error instantiating feemgr")
		return nil, fmt.Errorf("error instantiating feemgr binding: %w", err)
	}

	callOpts := &bind.CallOpts{
		Pending: false,
		Context: ctx,
	}
	return fm.Funds(callOpts, palomaAddress)
}

func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.conn.SuggestGasPrice(ctx)
}
