package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/VolumeFi/whoops"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	skyway "github.com/palomachain/paloma/v2/x/skyway/types"
	"github.com/palomachain/pigeon/chain"
	pigeonerrors "github.com/palomachain/pigeon/errors"
	"github.com/palomachain/pigeon/internal/queue"
	"github.com/palomachain/pigeon/util/slice"
	log "github.com/sirupsen/logrus"
)

type Processor struct {
	compass          *compass
	evmClient        *Client
	chainType        string
	chainReferenceID string

	turnstoneEVMContract common.Address //nolint:unused

	blockHeight       int64
	blockHeightHash   common.Hash
	minOnChainBalance *big.Int
}

func (p Processor) GetChainReferenceID() string {
	return p.chainReferenceID
}

func (p Processor) SupportedQueues() []string {
	return slice.Map(
		[]string{
			queue.QueueSuffixTurnstone,
			queue.QueueSuffixValidatorsBalances,
			queue.QueueSuffixReferenceBlock,
		},
		func(q string) string {
			return fmt.Sprintf("%s/%s/%s", p.chainType, p.chainReferenceID, q)
		},
	)
}

func (p Processor) SignMessages(ctx context.Context, messages ...chain.QueuedMessage) ([]chain.SignedQueuedMessage, error) {
	return slice.MapErr(messages, func(msg chain.QueuedMessage) (chain.SignedQueuedMessage, error) {
		msgBytes := crypto.Keccak256(
			append(
				[]byte(SignedMessagePrefix),
				msg.BytesToSign...,
			),
		)
		sig, err := p.evmClient.sign(ctx, msgBytes)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"message-id": msg.ID,
			}).Error("signing a message failed")
			return chain.SignedQueuedMessage{}, err
		}

		log.WithFields(log.Fields{
			"message-id": msg.ID,
		}).Info("signed a message")

		return chain.SignedQueuedMessage{
			QueuedMessage:   msg,
			Signature:       sig,
			SignedByAddress: p.evmClient.addr.Hex(),
		}, nil
	},
	)
}

func (p Processor) SkywaySignBatches(ctx context.Context, batches ...skyway.OutgoingTxBatch) ([]chain.SignedSkywayOutgoingTxBatch, error) {
	return slice.MapErr(batches, func(batch skyway.OutgoingTxBatch) (chain.SignedSkywayOutgoingTxBatch, error) {
		logger := log.WithFields(log.Fields{
			"batch-nonce": batch.BatchNonce,
		})

		msgBytes := crypto.Keccak256(
			append(
				[]byte(SignedMessagePrefix),
				batch.GetBytesToSign()...,
			),
		)
		sig, err := p.evmClient.sign(ctx, msgBytes)
		if err != nil {
			logger.WithError(err).Error("signing a batch failed")
			return chain.SignedSkywayOutgoingTxBatch{}, err
		}

		logger.Info("signed a batch")

		if err != nil {
			return chain.SignedSkywayOutgoingTxBatch{}, err
		}

		return chain.SignedSkywayOutgoingTxBatch{
			OutgoingTxBatch: batch,
			Signature:       sig,
			SignedByAddress: p.evmClient.addr.Hex(),
		}, nil
	},
	)
}

func (p Processor) ProcessMessages(ctx context.Context, queueTypeName queue.TypeName, msgs []chain.MessageWithSignatures) error {
	if !queueTypeName.IsTurnstoneQueue() {
		return chain.ErrProcessorDoesNotSupportThisQueue.Format(queueTypeName)
	}

	_, err := p.compass.processMessages(
		ctx,
		queueTypeName.String(),
		msgs,
		callOptions{},
	)
	return err
}

func (p Processor) EstimateMessages(ctx context.Context, queueTypeName queue.TypeName, msgs []chain.MessageWithSignatures) ([]chain.MessageWithEstimate, error) {
	if !queueTypeName.IsTurnstoneQueue() {
		return nil, chain.ErrProcessorDoesNotSupportThisQueue.Format(queueTypeName)
	}

	txs, err := p.compass.processMessages(
		ctx,
		queueTypeName.String(),
		msgs,
		callOptions{
			estimateOnly: true,
		},
	)
	if err != nil {
		// Even if we fail to estimate messages, we must continue, so we just
		// log the error
		// If we fail to estimate them all, `txs` will be empty, and nothing
		// will change anyway
		log.WithField("error", err).Warn("Failed to estimate messages")
	}

	res := make([]chain.MessageWithEstimate, 0, len(txs))
	for i, tx := range txs {
		if tx == nil {
			// We couldn't estimate this message, so we just ignore it
			continue
		}

		res = append(res, chain.MessageWithEstimate{
			MessageWithSignatures: msgs[i],
			Estimate:              tx.Gas(),
			EstimatedByAddress:    p.evmClient.addr.Hex(),
		})
	}

	return res, nil
}

func (p Processor) SkywayRelayBatches(ctx context.Context, batches []chain.SkywayBatchWithSignatures) error {
	return p.compass.skywayRelayBatches(
		ctx,
		batches,
	)
}

func (p Processor) SkywayEstimateBatches(ctx context.Context, batches []chain.SkywayBatchWithSignatures) ([]chain.EstimatedSkywayBatch, error) {
	res := make([]chain.EstimatedSkywayBatch, 0, len(batches))
	estimates, err := p.compass.skywayEstimateBatches(
		ctx,
		batches,
	)
	if err != nil {
		return nil, fmt.Errorf("processor::SkywayEstimateBatches: %w", err)
	}

	if len(estimates) != len(batches) {
		return nil, fmt.Errorf("processor::SkywayEstimateBatches: estimated %d batches, but got %d", len(batches), len(estimates))
	}

	for i, estimate := range estimates {
		res = append(res, chain.EstimatedSkywayBatch{
			OutgoingTxBatch:    batches[i].OutgoingTxBatch,
			EstimatedByAddress: p.evmClient.addr.Hex(),
			Value:              estimate,
		})
	}

	return res, nil
}

func (p Processor) ProvideEvidence(ctx context.Context, queueTypeName queue.TypeName, msgs []chain.MessageWithSignatures) error {
	if queueTypeName.IsValidatorsValancesQueue() {
		return p.compass.provideEvidenceForValidatorBalance(
			ctx,
			queueTypeName.String(),
			msgs,
		)
	}

	if queueTypeName.IsReferenceBlockQueue() {
		return p.compass.provideEvidenceForReferenceBlock(
			ctx,
			queueTypeName.String(),
			msgs,
		)
	}

	if !queueTypeName.IsTurnstoneQueue() {
		return chain.ErrProcessorDoesNotSupportThisQueue.Format(queueTypeName)
	}

	var gErr whoops.Group
	logger := log.WithField("queue-type-name", queueTypeName)
	for _, rawMsg := range msgs {
		if ctx.Err() != nil {
			logger.Debug("exiting processing message context")
			break
		}

		logger = logger.WithField("message-id", rawMsg.ID)
		switch {
		case len(rawMsg.ErrorData) > 0:
			logger.Debug("providing error proof for message")
			gErr.Add(
				p.compass.provideErrorProof(ctx, queueTypeName.String(), rawMsg),
			)
		case len(rawMsg.PublicAccessData) > 0:
			logger.Debug("providing tx proof for message")
			gErr.Add(
				p.compass.provideTxProof(ctx, queueTypeName.String(), rawMsg),
			)
		default:
			logger.Debug("skipping message as there is no proof")
			continue
		}
	}
	return gErr.Return()
}

func (p Processor) ExternalAccount() chain.ExternalAccount {
	return chain.ExternalAccount{
		ChainType:        p.chainType,
		ChainReferenceID: p.chainReferenceID,
		Address:          p.evmClient.addr.Hex(),
		PubKey:           p.evmClient.addr.Bytes(),
	}
}

func (p Processor) IsRightChain(ctx context.Context) error {
	block, err := p.evmClient.BlockByHash(ctx, p.blockHeightHash)
	if err != nil {
		return fmt.Errorf("BlockByHash: %w", err)
	}

	if p.chainReferenceID == "kava-main" && p.blockHeight == 5690000 {
		return p.isRightChain(common.HexToHash("0x76966b1d12b21d3ff22578948b05a42b3da5766fcc4b17ea48da5a154c80f08b"))
	}

	if p.chainReferenceID == "kava-main" && p.blockHeight == 5874963 {
		return p.isRightChain(common.HexToHash("0xd91fdced0e798342f47fd503d376f9665fb29bf468d7d4cc73bf9f470a9b0d76"))
	}

	return p.isRightChain(block.Hash())
}

func (p Processor) isRightChain(blockHash common.Hash) error {
	if blockHash != p.blockHeightHash {
		return pigeonerrors.Unrecoverable(chain.ErrNotConnectedToRightChain.WrapS(
			"chain %s hash at block height %d should be %s, while it is %s. Check the rpc-url of the chain in the config.",
			p.chainReferenceID,
			p.blockHeight,
			p.blockHeightHash,
			blockHash,
		))
	}

	return nil
}

func (p Processor) GetSkywayEvents(ctx context.Context, orchestrator string) ([]chain.SkywayEventer, error) {
	return p.compass.GetSkywayEvents(ctx, orchestrator)
}

// Submit all gathered events to Paloma. Events need to be sent in order to
// preserve skyway nonce order.
func (p Processor) SubmitEventClaims(
	ctx context.Context,
	events []chain.SkywayEventer,
	orchestrator string,
) error {
	var err error

	log.Debug("Submitting event claims")

	for _, event := range events {
		switch evt := event.(type) {
		case chain.BatchSendEvent:
			log.Debug("Submitting batch send event")
			err = p.compass.submitBatchSendToEVMClaim(ctx, evt, orchestrator)
		case chain.SendToPalomaEvent:
			log.Debug("Submitting send to paloma event")
			err = p.compass.submitSendToPalomaClaim(ctx, evt, orchestrator)
		case chain.LightNodeSaleEvent:
			log.Debug("Submitting light node sale event")
			err = p.compass.submitLightNodeSaleClaim(ctx, evt, orchestrator)
		default:
			err = errors.New("unknown event type")
		}

		if err != nil {
			log.WithError(err).Warn("Failed to submit claims")
			return err
		}
	}

	return nil
}
