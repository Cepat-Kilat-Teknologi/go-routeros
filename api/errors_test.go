package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceError_Error(t *testing.T) {
	err := &DeviceError{Category: 1, Message: "bad argument"}
	assert.Equal(t, "routeros: trap: bad argument (category 1)", err.Error())
}

func TestDeviceError_TypeAssertion(t *testing.T) {
	var err error = &DeviceError{Category: 0, Message: "missing"}
	de, ok := err.(*DeviceError)
	assert.True(t, ok)
	assert.Equal(t, 0, de.Category)
}

func TestFatalError_Error(t *testing.T) {
	err := &FatalError{Message: "session terminated"}
	assert.Equal(t, "routeros: fatal: session terminated", err.Error())
}

func TestFatalError_TypeAssertion(t *testing.T) {
	var err error = &FatalError{Message: "gone"}
	fe, ok := err.(*FatalError)
	assert.True(t, ok)
	assert.Equal(t, "gone", fe.Message)
}
