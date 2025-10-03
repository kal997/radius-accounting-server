//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/kal997/radius-accounting-server/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLogger_Integration(t *testing.T) {
	tmpFile := "/tmp/test_logger.log"
	defer os.Remove(tmpFile)

	logger, err := logger.NewFileLogger(tmpFile)
	require.NoError(t, err)
	defer logger.Close()

	err = logger.Log(context.Background(), "test message")
	assert.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}
