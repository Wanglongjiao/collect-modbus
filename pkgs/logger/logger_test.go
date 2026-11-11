package logger

import (
	"errors"
	"testing"
)

func TestJSONLogger(t *testing.T) {
	logger, err := NewLogger("test",
		WithField("defined_key", "defined_value"),
		WithInfoLevel(),
		// WithFileRotationP("log/out.log", 10, 3, 5),
		// WithDisableConsole(),
		WithJsonFormat(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Sync()

	err = errors.New("pkg error")
	logger.Error("err occurs", WrapMeta(nil, NewMeta("para1", "value1"), NewMeta("para2", "value2"))...)
	logger.Error("err occurs", WrapMeta(err, NewMeta("para1", "value1"), NewMeta("para2", "value2"))...)
	logger.Info("info msg")
	logger.Debug("debug msg")
	Get("test").Info("dddddddddddddd")
}

func TestTextLogger(t *testing.T) {
	logger, err := NewLogger("test",
		WithInfoLevel(),
		// WithFileRotationP("log/out.log", 10, 3, 5),
		// WithDisableConsole(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Sync()
	logger.Error("err occurs")
	logger.Error("err occurs")
	logger.Info("info msg")
	logger.Debug("debug msg")
	Get("test").Info("dddddddddddddd")
	Get("test").Sugar().Info("dddddddfffdddddd")
}

func BenchmarkJsonLogger(b *testing.B) {
	b.ResetTimer()
	logger, err := NewLogger("test")
	if err != nil {
		b.Fatal(err)
	}
	defer logger.Sync()
}
