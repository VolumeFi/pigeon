// Code generated by mockery v2.11.0. DO NOT EDIT.

package evm

import (
	context "context"
	testing "testing"

	mock "github.com/stretchr/testify/mock"

	types "github.com/palomachain/pigeon/types/paloma/x/evm/types"
)

// mockPalomaClienter is an autogenerated mock type for the palomaClienter type
type mockPalomaClienter struct {
	mock.Mock
}

// DeleteJob provides a mock function with given fields: ctx, queueTypeName, id
func (_m *mockPalomaClienter) DeleteJob(ctx context.Context, queueTypeName string, id uint64) error {
	ret := _m.Called(ctx, queueTypeName, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, uint64) error); ok {
		r0 = rf(ctx, queueTypeName, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// QueryGetEVMValsetByID provides a mock function with given fields: ctx, id, chainID
func (_m *mockPalomaClienter) QueryGetEVMValsetByID(ctx context.Context, id uint64, chainID string) (*types.Valset, error) {
	ret := _m.Called(ctx, id, chainID)

	var r0 *types.Valset
	if rf, ok := ret.Get(0).(func(context.Context, uint64, string) *types.Valset); ok {
		r0 = rf(ctx, id, chainID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Valset)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, uint64, string) error); ok {
		r1 = rf(ctx, id, chainID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockPalomaClienter creates a new instance of mockPalomaClienter. It also registers a cleanup function to assert the mocks expectations.
func newMockPalomaClienter(t testing.TB) *mockPalomaClienter {
	mock := &mockPalomaClienter{}

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}