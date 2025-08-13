package audiocd

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	buf := bytes.Buffer{}
	logger := log.New(&buf, "cdda:", 0)
	cdr, _ := OpenDefaultL(LogModeLogger, logger)

	assert.Greater(t, buf.Len(), 0)
	str := buf.String()

	t.Log(str)
	assert.True(t, strings.HasPrefix(str, "cdda:"))

	if cdr != nil {
		assert.Equal(t, LogModeLogger, cdr.LogMode)
		assert.Equal(t, logger, cdr.Logger)
	}
}
