package paloma

import (
	"context"

	"github.com/VolumeFi/whoops"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/cosmos/gogoproto/proto"
	consensus "github.com/palomachain/paloma/v2/x/consensus/types"
	valset "github.com/palomachain/paloma/v2/x/valset/types"
	"github.com/palomachain/pigeon/chain"
	"github.com/palomachain/pigeon/config"
	"github.com/palomachain/pigeon/util/ion"
	"github.com/palomachain/pigeon/util/slice"
)

//go:generate mockery --name=MessageSender
type MessageSender interface {
	SendMsg(ctx context.Context, msg sdk.Msg, memo string, opts ...ion.SendMsgOption) (*sdk.TxResponse, error)
}

//go:generate mockery --name=IonClient
type IonClient interface {
	Status(context.Context) (*coretypes.ResultStatus, error)
	DecodeBech32ValAddr(string) (sdk.ValAddress, error)
	GetKeybase() keyring.Keyring
	SetSDKContext() func()
	GetKeyAddress() (sdk.AccAddress, error)
}

type Client struct {
	PalomaConfig config.Paloma // This is only needed ONCE ! Can we remove it?
	GRPCClient   grpc.ClientConn

	ic            IonClient
	unpacker      codectypes.AnyUnpacker
	messageSender MessageSender
	sendingOpts   []ion.SendMsgOption

	creator        string
	creatorValoper string
	valAddr        sdk.ValAddress
}

func NewClient(cfg config.Paloma, grpcWrapper grpc.ClientConn, ion IonClient, sender MessageSender, unpacker codectypes.AnyUnpacker) *Client {
	return (&Client{
		PalomaConfig:  cfg,
		GRPCClient:    grpcWrapper,
		ic:            ion,
		unpacker:      unpacker,
		messageSender: sender,
	}).init()
}

func (c *Client) init() *Client {
	c.creator = getCreator(c)
	c.creatorValoper = getCreatorAsValoper(c)
	c.valAddr = sdk.ValAddress(getValidatorAddress(c).Bytes())
	c.sendingOpts = []ion.SendMsgOption{ion.WithFeeGranter(c.valAddr.Bytes())}
	return c
}

type BroadcastMessageSignatureIn struct {
	ID              uint64
	QueueTypeName   string
	Signature       []byte
	SignedByAddress string
}

// BroadcastMessageSignatures takes a list of signatures that need to be sent over to the chain.
// It build the message and sends it over.
func (c *Client) BroadcastMessageSignatures(ctx context.Context, signatures ...BroadcastMessageSignatureIn) error {
	return broadcastMessageSignatures(ctx, c.messageSender, c.creator, c.sendingOpts, signatures...)
}

func (c *Client) BlockHeight(ctx context.Context) (int64, error) {
	res, err := c.ic.Status(ctx)
	if err != nil {
		return 0, err
	}

	if res.SyncInfo.CatchingUp {
		return 0, ErrNodeIsNotInSync
	}

	return res.SyncInfo.LatestBlockHeight, nil
}

type ChainInfoIn struct {
	ChainType        string
	ChainReferenceID string
	AccAddress       string
	PubKey           []byte
	Traits           []string
}

// AddExternalChainInfo adds info about the external chain. It adds the chain's
// account addresses that the pigeon knows about.
func (c *Client) AddExternalChainInfo(ctx context.Context, chainInfos ...ChainInfoIn) error {
	if len(chainInfos) == 0 {
		return nil
	}

	msg := &valset.MsgAddExternalChainInfoForValidator{
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	msg.ChainInfos = slice.Map(
		chainInfos,
		func(in ChainInfoIn) *valset.ExternalChainInfo {
			return &valset.ExternalChainInfo{
				ChainType:        in.ChainType,
				ChainReferenceID: in.ChainReferenceID,
				Address:          in.AccAddress,
				Pubkey:           in.PubKey,
				Traits:           in.Traits,
			}
		},
	)

	_, err := c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) AddMessageEvidence(ctx context.Context, queueTypeName string, messageID uint64, proof proto.Message) error {
	anyProof, err := codectypes.NewAnyWithValue(proof)
	if err != nil {
		return err
	}
	msg := &consensus.MsgAddEvidence{
		Proof:         anyProof,
		MessageID:     messageID,
		QueueTypeName: queueTypeName,
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	_, err = c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) AddMessagesGasEstimate(ctx context.Context, queueTypeName string, msgs ...chain.MessageWithEstimate) error {
	estimates := make([]*consensus.MsgAddMessageGasEstimates_GasEstimate, len(msgs))
	for i, msg := range msgs {
		estimates[i] = &consensus.MsgAddMessageGasEstimates_GasEstimate{
			MsgId:              msg.ID,
			QueueTypeName:      queueTypeName,
			Value:              msg.Estimate,
			EstimatedByAddress: msg.EstimatedByAddress,
		}
	}
	msg := &consensus.MsgAddMessageGasEstimates{
		Estimates: estimates,
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	_, err := c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) SetPublicAccessData(ctx context.Context, queueTypeName string, messageID, valsetID uint64, data []byte) error {
	msg := &consensus.MsgSetPublicAccessData{
		Data:          data,
		ValsetID:      valsetID,
		MessageID:     messageID,
		QueueTypeName: queueTypeName,
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	_, err := c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) SetErrorData(ctx context.Context, queueTypeName string, messageID uint64, data []byte) error {
	msg := &consensus.MsgSetErrorData{
		Data:          data,
		MessageID:     messageID,
		QueueTypeName: queueTypeName,
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	_, err := c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) KeepValidatorAlive(ctx context.Context, appVersion string) error {
	msg := &valset.MsgKeepAlive{
		PigeonVersion: appVersion,
		Metadata: valset.MsgMetadata{
			Creator: c.creator,
		},
	}

	_, err := c.messageSender.SendMsg(ctx, msg, "", c.sendingOpts...)
	return err
}

func (c *Client) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	res, err := c.ic.Status(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) PalomaStatus(ctx context.Context) error {
	res, err := c.Status(ctx)

	if IsPalomaDown(err) {
		return whoops.Wrap(ErrPalomaIsDown, err)
	}

	if err != nil {
		return err
	}
	_ = res
	return nil
}

func (c *Client) GetValidator(ctx context.Context) (*stakingtypes.Validator, error) {
	address := c.GetValidatorAddress().String()
	_, err := c.ic.DecodeBech32ValAddr(address)
	if err != nil {
		return nil, err
	}
	queryClient := stakingtypes.NewQueryClient(c.GRPCClient)
	req := &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: address,
	}
	res, err := queryClient.Validator(ctx, req)
	if err != nil {
		return nil, err
	}
	return &res.Validator, nil
}

func broadcastMessageSignatures(
	ctx context.Context,
	ms MessageSender,
	creator string,
	opts []ion.SendMsgOption,
	signatures ...BroadcastMessageSignatureIn,
) error {
	if len(signatures) == 0 {
		return nil
	}
	signedMessages := make([]*consensus.ConsensusMessageSignature, len(signatures))
	for i, sig := range signatures {
		signedMessages[i] = &consensus.ConsensusMessageSignature{
			Id:              sig.ID,
			QueueTypeName:   sig.QueueTypeName,
			Signature:       sig.Signature,
			SignedByAddress: sig.SignedByAddress,
		}
	}
	msg := &consensus.MsgAddMessagesSignatures{
		SignedMessages: signedMessages,
		Metadata: valset.MsgMetadata{
			Creator: creator,
		},
	}
	_, err := ms.SendMsg(ctx, msg, "", opts...)
	return err
}

func (c *Client) Keyring() keyring.Keyring {
	return c.ic.GetKeybase()
}

func (c *Client) GetValidatorAddress() sdk.ValAddress {
	return c.valAddr
}

func (c *Client) GetCreator() string {
	return c.creator
}

func (c *Client) GetSigner() string {
	addr, err := c.ic.GetKeyAddress()
	if err != nil {
		panic(err)
	}

	return addr.String()
}

func getValidatorAddress(c *Client) sdk.Address {
	address, err := getKeyAddress(c, c.PalomaConfig.ValidatorKey)
	if err != nil {
		panic(err)
	}
	return address
}

func getKeyAddress(c *Client, keyname string) (sdk.Address, error) {
	key, err := c.Keyring().Key(keyname)
	if err != nil {
		return nil, err
	}
	address, err := key.GetAddress()
	if err != nil {
		return nil, err
	}
	return address, nil
}

// TODO: CLEAN THIS UP???
func getCreator(c *Client) string {
	return c.addressString(getValidatorAddress(c))
}

func getCreatorAsValoper(c *Client) string {
	return c.addressString(sdk.ValAddress(getValidatorAddress(c).Bytes()))
}

func (c Client) addressString(val sdk.Address) string {
	defer (c.ic.SetSDKContext())()
	return val.String()
}
