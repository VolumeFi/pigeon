// Code generated by mockery v2.46.2. DO NOT EDIT.

package evm

import (
	big "math/big"

	abi "github.com/ethereum/go-ethereum/accounts/abi"

	common "github.com/ethereum/go-ethereum/common"

	context "context"

	ethereum "github.com/ethereum/go-ethereum"

	mock "github.com/stretchr/testify/mock"

	time "time"

	types "github.com/ethereum/go-ethereum/core/types"
)

// mockEvmClienter is an autogenerated mock type for the evmClienter type
type mockEvmClienter struct {
	mock.Mock
}

// BalanceAt provides a mock function with given fields: ctx, address, blockHeight
func (_m *mockEvmClienter) BalanceAt(ctx context.Context, address common.Address, blockHeight uint64) (*big.Int, error) {
	ret := _m.Called(ctx, address, blockHeight)

	if len(ret) == 0 {
		panic("no return value specified for BalanceAt")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, uint64) (*big.Int, error)); ok {
		return rf(ctx, address, blockHeight)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, uint64) *big.Int); ok {
		r0 = rf(ctx, address, blockHeight)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Address, uint64) error); ok {
		r1 = rf(ctx, address, blockHeight)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeployContract provides a mock function with given fields: ctx, chainID, rawABI, bytecode, constructorInput
func (_m *mockEvmClienter) DeployContract(ctx context.Context, chainID *big.Int, rawABI string, bytecode []byte, constructorInput []byte) (common.Address, *types.Transaction, error) {
	ret := _m.Called(ctx, chainID, rawABI, bytecode, constructorInput)

	if len(ret) == 0 {
		panic("no return value specified for DeployContract")
	}

	var r0 common.Address
	var r1 *types.Transaction
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int, string, []byte, []byte) (common.Address, *types.Transaction, error)); ok {
		return rf(ctx, chainID, rawABI, bytecode, constructorInput)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int, string, []byte, []byte) common.Address); ok {
		r0 = rf(ctx, chainID, rawABI, bytecode, constructorInput)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *big.Int, string, []byte, []byte) *types.Transaction); ok {
		r1 = rf(ctx, chainID, rawABI, bytecode, constructorInput)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*types.Transaction)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, *big.Int, string, []byte, []byte) error); ok {
		r2 = rf(ctx, chainID, rawABI, bytecode, constructorInput)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ExecuteSmartContract provides a mock function with given fields: ctx, chainID, contractAbi, addr, opts, method, arguments, gasEstimate
func (_m *mockEvmClienter) ExecuteSmartContract(ctx context.Context, chainID *big.Int, contractAbi abi.ABI, addr common.Address, opts callOptions, method string, arguments []any, gasEstimate *big.Int) (*types.Transaction, error) {
	ret := _m.Called(ctx, chainID, contractAbi, addr, opts, method, arguments, gasEstimate)

	if len(ret) == 0 {
		panic("no return value specified for ExecuteSmartContract")
	}

	var r0 *types.Transaction
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int, abi.ABI, common.Address, callOptions, string, []any, *big.Int) (*types.Transaction, error)); ok {
		return rf(ctx, chainID, contractAbi, addr, opts, method, arguments, gasEstimate)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int, abi.ABI, common.Address, callOptions, string, []any, *big.Int) *types.Transaction); ok {
		r0 = rf(ctx, chainID, contractAbi, addr, opts, method, arguments, gasEstimate)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Transaction)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *big.Int, abi.ABI, common.Address, callOptions, string, []any, *big.Int) error); ok {
		r1 = rf(ctx, chainID, contractAbi, addr, opts, method, arguments, gasEstimate)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FilterLogs provides a mock function with given fields: ctx, fq, currBlockHeight, fn
func (_m *mockEvmClienter) FilterLogs(ctx context.Context, fq ethereum.FilterQuery, currBlockHeight *big.Int, fn func([]types.Log) bool) (bool, error) {
	ret := _m.Called(ctx, fq, currBlockHeight, fn)

	if len(ret) == 0 {
		panic("no return value specified for FilterLogs")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ethereum.FilterQuery, *big.Int, func([]types.Log) bool) (bool, error)); ok {
		return rf(ctx, fq, currBlockHeight, fn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ethereum.FilterQuery, *big.Int, func([]types.Log) bool) bool); ok {
		r0 = rf(ctx, fq, currBlockHeight, fn)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, ethereum.FilterQuery, *big.Int, func([]types.Log) bool) error); ok {
		r1 = rf(ctx, fq, currBlockHeight, fn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindBlockNearestToTime provides a mock function with given fields: ctx, startingHeight, when
func (_m *mockEvmClienter) FindBlockNearestToTime(ctx context.Context, startingHeight uint64, when time.Time) (uint64, error) {
	ret := _m.Called(ctx, startingHeight, when)

	if len(ret) == 0 {
		panic("no return value specified for FindBlockNearestToTime")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64, time.Time) (uint64, error)); ok {
		return rf(ctx, startingHeight, when)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64, time.Time) uint64); ok {
		r0 = rf(ctx, startingHeight, when)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64, time.Time) error); ok {
		r1 = rf(ctx, startingHeight, when)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindCurrentBlockNumber provides a mock function with given fields: ctx
func (_m *mockEvmClienter) FindCurrentBlockNumber(ctx context.Context) (*big.Int, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for FindCurrentBlockNumber")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*big.Int, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *big.Int); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetEthClient provides a mock function with given fields:
func (_m *mockEvmClienter) GetEthClient() ethClientConn {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetEthClient")
	}

	var r0 ethClientConn
	if rf, ok := ret.Get(0).(func() ethClientConn); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ethClientConn)
		}
	}

	return r0
}

// LastValsetID provides a mock function with given fields: ctx, addr
func (_m *mockEvmClienter) LastValsetID(ctx context.Context, addr common.Address) (*big.Int, error) {
	ret := _m.Called(ctx, addr)

	if len(ret) == 0 {
		panic("no return value specified for LastValsetID")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Address) (*big.Int, error)); ok {
		return rf(ctx, addr)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Address) *big.Int); ok {
		r0 = rf(ctx, addr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Address) error); ok {
		r1 = rf(ctx, addr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryUserFunds provides a mock function with given fields: ctx, feemgraddr, palomaAddress
func (_m *mockEvmClienter) QueryUserFunds(ctx context.Context, feemgraddr common.Address, palomaAddress [32]byte) (*big.Int, error) {
	ret := _m.Called(ctx, feemgraddr, palomaAddress)

	if len(ret) == 0 {
		panic("no return value specified for QueryUserFunds")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, [32]byte) (*big.Int, error)); ok {
		return rf(ctx, feemgraddr, palomaAddress)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Address, [32]byte) *big.Int); ok {
		r0 = rf(ctx, feemgraddr, palomaAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Address, [32]byte) error); ok {
		r1 = rf(ctx, feemgraddr, palomaAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SuggestGasPrice provides a mock function with given fields: ctx
func (_m *mockEvmClienter) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for SuggestGasPrice")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*big.Int, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *big.Int); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TransactionByHash provides a mock function with given fields: ctx, txHash
func (_m *mockEvmClienter) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	ret := _m.Called(ctx, txHash)

	if len(ret) == 0 {
		panic("no return value specified for TransactionByHash")
	}

	var r0 *types.Transaction
	var r1 bool
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) (*types.Transaction, bool, error)); ok {
		return rf(ctx, txHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Transaction); ok {
		r0 = rf(ctx, txHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Transaction)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) bool); ok {
		r1 = rf(ctx, txHash)
	} else {
		r1 = ret.Get(1).(bool)
	}

	if rf, ok := ret.Get(2).(func(context.Context, common.Hash) error); ok {
		r2 = rf(ctx, txHash)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// TransactionReceipt provides a mock function with given fields: ctx, txHash
func (_m *mockEvmClienter) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	ret := _m.Called(ctx, txHash)

	if len(ret) == 0 {
		panic("no return value specified for TransactionReceipt")
	}

	var r0 *types.Receipt
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) (*types.Receipt, error)); ok {
		return rf(ctx, txHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) *types.Receipt); ok {
		r0 = rf(ctx, txHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Receipt)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(ctx, txHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockEvmClienter creates a new instance of mockEvmClienter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockEvmClienter(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockEvmClienter {
	mock := &mockEvmClienter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
