package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, "distillery", NAME)
	assert.Equal(t, "v1.0.0", SUMMARY)
}
