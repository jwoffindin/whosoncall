package main

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	assert := require.New(t)

	l := logrus.NewEntry(logrus.StandardLogger())

	oncall, err := whosOnCall(l, "P538IZH")
	assert.NoError(err)
	assert.Equal(&onCallUserResponse{}, oncall)
}
