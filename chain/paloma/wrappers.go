package paloma

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/VolumeFi/whoops"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/grpc"
	vtypes "github.com/palomachain/paloma/v2/x/valset/types"
	"github.com/palomachain/pigeon/internal/liblog"
	"github.com/palomachain/pigeon/util/ion"
	ggrpc "google.golang.org/grpc"
)

var _ grpc.ClientConn = GRPCClientWrapper{}

var _ MessageSender = &PalomaMessageSender{}

type GRPCClientWrapper struct {
	W grpc.ClientConn
}

type KeyRotator interface {
	RotateKeys(context.Context)
}

type PalomaMessageSender struct {
	R          KeyRotator
	W          MessageSender
	GetCreator func() string
	GetSigner  func() string
	m          *sync.Mutex
}

func (g GRPCClientWrapper) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...ggrpc.CallOption) error {
	err := g.W.Invoke(ctx, method, args, reply, opts...)
	if IsPalomaDown(err) {
		return whoops.Wrap(ErrPalomaIsDown, err)
	}
	return err
}

func (g GRPCClientWrapper) NewStream(ctx context.Context, desc *ggrpc.StreamDesc, method string, opts ...ggrpc.CallOption) (ggrpc.ClientStream, error) {
	stream, err := g.W.NewStream(ctx, desc, method, opts...)

	if IsPalomaDown(err) {
		return nil, whoops.Wrap(ErrPalomaIsDown, err)
	}

	return stream, err
}

func NewPalomaMessageSender(R KeyRotator, W MessageSender) *PalomaMessageSender {
	return &PalomaMessageSender{
		R: R,
		W: W,
		m: &sync.Mutex{},
	}
}

func (m *PalomaMessageSender) WithCreatorProvider(p func() string) *PalomaMessageSender {
	m.GetCreator = p
	return m
}

func (m *PalomaMessageSender) WithSignerProvider(p func() string) *PalomaMessageSender {
	m.GetSigner = p
	return m
}

func (m *PalomaMessageSender) SendMsg(ctx context.Context, msg sdk.Msg, memo string, opts ...ion.SendMsgOption) (*sdk.TxResponse, error) {
	logger := liblog.WithContext(ctx).WithField("component", "message-sender")

	// It's theoretically possible two messages would be sent in parallel by different consumers of the client.
	// In order to avoid a race conditions where keys are being rotated and then used twice, we add some basic locking here.
	m.m.Lock()
	defer m.m.Unlock()

	m.R.RotateKeys(ctx)
	creator := m.GetCreator()
	signer := m.GetSigner()

	logger.WithField("creator", creator).WithField("signer", signer).Debug("Injecting metadata")
	if err := tryInjectMetadata(msg, vtypes.MsgMetadata{
		Creator: m.GetCreator(),
		Signers: []string{signer},
	}); err != nil {
		return nil, fmt.Errorf("failed to inject metadata: %w", err)
	}

	logger.WithField("msg", msg.String()).Debug("Sending message...")
	res, err := m.W.SendMsg(ctx, msg, memo, opts...)
	if IsPalomaDown(err) {
		return nil, whoops.Wrap(ErrPalomaIsDown, err)
	}

	return res, err
}

func tryInjectMetadata(msg sdk.Msg, md vtypes.MsgMetadata) error {
	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("%v is not a struct", msg)
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := field.Type()

		// Check if the field's type matches the target type
		if fieldType == reflect.TypeOf(vtypes.MsgMetadata{}) {
			field.Set(reflect.ValueOf(md))
		}
	}

	return nil
}
