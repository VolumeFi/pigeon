package chain

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	skyway "github.com/palomachain/paloma/v2/x/skyway/types"
	"github.com/palomachain/pigeon/health"
	"github.com/palomachain/pigeon/internal/queue"
)

type QueuedMessage struct {
	ID               uint64
	Nonce            []byte
	BytesToSign      []byte
	PublicAccessData []byte
	ErrorData        []byte
	Msg              any
	Estimate         *big.Int
}

type SignedQueuedMessage struct {
	QueuedMessage
	Signature       []byte
	SignedByAddress string
}

type SignedSkywayOutgoingTxBatch struct {
	skyway.OutgoingTxBatch
	Signature       []byte
	SignedByAddress string
}

type MessageToProcess struct {
	QueueTypeName string
	Msg           QueuedMessage
}

type ValidatorSignature struct {
	ValAddress      sdk.ValAddress
	Signature       []byte
	SignedByAddress string
	PublicKey       []byte
}

type SignedEntity interface {
	GetSignatures() []ValidatorSignature
	GetBytes() []byte
}

type MessageWithSignatures struct {
	QueuedMessage
	Signatures []ValidatorSignature
}

type MessageWithEstimate struct {
	MessageWithSignatures
	Estimate           uint64
	EstimatedByAddress string
}

func (msg MessageWithSignatures) GetSignatures() []ValidatorSignature {
	return msg.Signatures
}

func (msg MessageWithSignatures) GetBytes() []byte {
	return msg.BytesToSign
}

type SkywayBatchWithSignatures struct {
	skyway.OutgoingTxBatch
	Signatures []ValidatorSignature
}

type EstimatedSkywayBatch struct {
	skyway.OutgoingTxBatch
	EstimatedByAddress string
	Value              uint64
}

func (gb SkywayBatchWithSignatures) GetSignatures() []ValidatorSignature {
	return gb.Signatures
}

func (gb SkywayBatchWithSignatures) GetBytes() []byte {
	return gb.GetBytesToSign()
}

type ExternalAccount struct {
	ChainType        string
	ChainReferenceID string

	Address string
	PubKey  []byte
}

type SkywayEventer interface {
	GetEventNonce() uint64
	GetSkywayNonce() uint64
}

type BatchSendEvent struct {
	TokenContract  string
	EthBlockHeight uint64
	EventNonce     uint64
	BatchNonce     uint64
	SkywayNonce    uint64
}

func (e BatchSendEvent) GetEventNonce() uint64  { return e.EventNonce }
func (e BatchSendEvent) GetSkywayNonce() uint64 { return e.SkywayNonce }

type SendToPalomaEvent struct {
	EthereumSender string
	PalomaReceiver string
	TokenContract  string
	EthBlockHeight uint64
	EventNonce     uint64
	Amount         uint64
	SkywayNonce    uint64
}

func (e SendToPalomaEvent) GetEventNonce() uint64  { return e.EventNonce }
func (e SendToPalomaEvent) GetSkywayNonce() uint64 { return e.SkywayNonce }

type LightNodeSaleEvent struct {
	ClientAddress        string
	SmartContractAddress string
	Amount               uint64
	EthBlockHeight       uint64
	EventNonce           uint64
	SkywayNonce          uint64
}

func (e LightNodeSaleEvent) GetEventNonce() uint64  { return e.EventNonce }
func (e LightNodeSaleEvent) GetSkywayNonce() uint64 { return e.SkywayNonce }

type ChainInfo interface {
	ChainReferenceID() string
	ChainID() string
	ChainType() string
}

//go:generate mockery --name=Processor
type Processor interface {
	health.Checker
	// GetChainReferenceID returns the chain reference EventNonce against which the processor is running.
	GetChainReferenceID() string

	// SupportedQueues is a list of consensus queues that this processor supports and expects to work with.
	SupportedQueues() []string

	ExternalAccount() ExternalAccount

	// SignMessages takes a list of messages and signs them via their key.
	SignMessages(ctx context.Context, messages ...QueuedMessage) ([]SignedQueuedMessage, error)

	// ProcessMessages will receive messages from the current queues and it's on the implementation
	// to ensure that there are enough signatures for consensus.
	ProcessMessages(context.Context, queue.TypeName, []MessageWithSignatures) error

	EstimateMessages(context.Context, queue.TypeName, []MessageWithSignatures) ([]MessageWithEstimate, error)

	// ProvideEvidence takes a queue name and a list of messages that have already been executed. This
	// takes the "public evidence" from the message and gets the information back to the Paloma.
	ProvideEvidence(context.Context, queue.TypeName, []MessageWithSignatures) error

	SubmitEventClaims(context.Context, []SkywayEventer, string) error

	// it verifies if it's being connected to the right chain
	IsRightChain(ctx context.Context) error

	SkywaySignBatches(context.Context, ...skyway.OutgoingTxBatch) ([]SignedSkywayOutgoingTxBatch, error)
	SkywayEstimateBatches(context.Context, []SkywayBatchWithSignatures) ([]EstimatedSkywayBatch, error)
	SkywayRelayBatches(context.Context, []SkywayBatchWithSignatures) error

	GetSkywayEvents(context.Context, string) ([]SkywayEventer, error)
}

type ProcessorBuilder interface {
	Build(ChainInfo) (Processor, error)
}
