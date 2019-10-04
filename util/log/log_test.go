package log

import "testing"

func TestGetLogger(t *testing.T) {
	testLogger := GetLogger()
	testLogger.Debug("Debug test")
	testLogger.Trace("Trace test")
	testLogger.Info("Info test")
	testLogger.Warn("Warn test")
	testLogger.Error("Error test")
}
