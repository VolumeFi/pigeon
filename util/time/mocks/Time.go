// Code generated by mockery v2.35.4. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// Time is an autogenerated mock type for the Time type
type Time struct {
	mock.Mock
}

// Now provides a mock function with given fields:
func (_m *Time) Now() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// NewTime creates a new instance of Time. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTime(t interface {
	mock.TestingT
	Cleanup(func())
},
) *Time {
	mock := &Time{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}