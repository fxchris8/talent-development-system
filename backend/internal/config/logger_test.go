package config

import (
	"testing"

	"github.com/sirupsen/logrus"
)

// With LOG_LEVEL unset the logger used to fall to level 0 (panic only), which silently
// dropped every log line in production, including fatal start-up messages.
func TestLoggerDefaultsToInfoWhenLogLevelIsUnset(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	log := NewLogger(NewViper())
	if log.GetLevel() != logrus.InfoLevel {
		t.Fatalf("default level = %v, want info", log.GetLevel())
	}
	if !log.IsLevelEnabled(logrus.FatalLevel) || !log.IsLevelEnabled(logrus.WarnLevel) {
		t.Fatal("fatal and warning messages must be visible by default")
	}
}

func TestLoggerHonoursExplicitLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "2")
	if got := NewLogger(NewViper()).GetLevel(); got != logrus.ErrorLevel {
		t.Fatalf("level = %v, want error", got)
	}
}
