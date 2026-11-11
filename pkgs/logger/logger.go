package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logMap sync.Map

const (
	// DefaultLevel the default log level
	defaultLevel = zapcore.InfoLevel

	// DefaultTimeLayout the default time layout;
	defaultTimeLayout = "2006-01-02T15:04:05.999"
)

// Option custom setup config
type Option func(*option)

type option struct {
	level          zapcore.Level
	fields         map[string]string
	file           io.Writer
	timeLayout     string
	levelLayout    string
	callerLayout   string
	disableConsole bool
	jsonFormat     bool
}

func WithLevel(l string) Option {
	level := zap.InfoLevel
	switch strings.ToLower(strings.TrimSpace(l)) {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}
	return func(opt *option) {
		opt.level = level
	}
}

// WithDebugLevel only greater than 'level' will output
func WithDebugLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.DebugLevel
	}
}

// WithInfoLevel only greater than 'level' will output
func WithInfoLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.InfoLevel
	}
}

// WithWarnLevel only greater than 'level' will output
func WithWarnLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.WarnLevel
	}
}

// WithErrorLevel only greater than 'level' will output
func WithErrorLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.ErrorLevel
	}
}

// WithField add some field(s) to log
func WithField(key, value string) Option {
	return func(opt *option) {
		opt.fields[key] = value
	}
}

// WithFileP write log to some file
func WithFileP(file string) Option {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0766); err != nil {
		panic(err)
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0766)
	if err != nil {
		panic(err)
	}

	return func(opt *option) {
		opt.file = zapcore.Lock(f)
	}
}

// WithFileRotationP write log to some file with rotation
func WithFileRotationP(file string, maxSize, maxBackups, maxAge int) Option {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0766); err != nil {
		panic(err)
	}

	return func(opt *option) {
		opt.file = &lumberjack.Logger{ // concurrent-safed
			Filename:   file,       // 文件路径
			MaxSize:    maxSize,    // 单个文件最大尺寸，默认单位 M
			MaxBackups: maxBackups, // 最多保留 300 个备份
			MaxAge:     maxAge,     // 最大时间，默认单位 day
			LocalTime:  true,       // 使用本地时间
			Compress:   true,       // 是否压缩 disabled by default
		}
	}
}

// WithTimeLayout custom time format
func WithTimeLayout(timeLayout string) Option {
	return func(opt *option) {
		opt.timeLayout = timeLayout
	}
}

// "xxx%sxxx"
func WithLevelLayout(levelLayout string) Option {
	return func(opt *option) {
		opt.levelLayout = levelLayout
	}
}

// "xxx%sxxx"
func WithCallerLayout(callerLayout string) Option {
	return func(opt *option) {
		opt.callerLayout = callerLayout
	}
}

// WithEnableConsole write log to os.Stdout or os.Stderr
func WithDisableConsole(ok bool) Option {
	return func(opt *option) {
		opt.disableConsole = ok
	}
}

func WithJsonFormat() Option {
	return func(opt *option) {
		opt.jsonFormat = true
	}
}

// NewJSONLogger return a json-encoder zap logger,
func NewLogger(key string, opts ...Option) (*zap.Logger, error) {
	opt := &option{level: defaultLevel, fields: make(map[string]string)}
	for _, f := range opts {
		f(opt)
	}

	// similar to zap.NewProductionEncoderConfig()
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger", // used by logger.Named(key); optional; useless
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace", // use by zap.AddStacktrace; optional; useless
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel: func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			if opt.levelLayout != "" {
				enc.AppendString(fmt.Sprintf(opt.levelLayout, l.String()))
			} else {
				enc.AppendString(l.String())
			}
		}, // 小写编码器
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			if opt.timeLayout != "" {
				enc.AppendString(t.Format(opt.timeLayout))
			} else {
				enc.AppendString(t.Format(defaultTimeLayout))
			}
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller: func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			if opt.callerLayout != "" {
				enc.AppendString(fmt.Sprintf(opt.callerLayout, caller.TrimmedPath()))
			} else {
				enc.AppendString(caller.TrimmedPath())
			}
		}, // 全路径编码器
		ConsoleSeparator: "\t",
	}
	var encoder zapcore.Encoder
	if opt.jsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewTee()
	stdout := zapcore.Lock(os.Stdout) // lock for concurrent safe
	stderr := zapcore.Lock(os.Stderr) // lock for concurrent safe
	if !opt.disableConsole {
		core = zapcore.NewTee(
			// lowPriority usd by info\debug\warn
			zapcore.NewCore(encoder,
				zapcore.NewMultiWriteSyncer(stdout),
				zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
					return lvl >= opt.level && lvl < zapcore.ErrorLevel
				}),
			),
			// highPriority usd by error\panic\fatal
			zapcore.NewCore(encoder,
				zapcore.NewMultiWriteSyncer(stderr),
				zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
					return lvl >= opt.level && lvl >= zapcore.ErrorLevel
				}),
			),
		)
	}

	if opt.file != nil {
		core = zapcore.NewTee(core,
			zapcore.NewCore(encoder,
				zapcore.AddSync(opt.file),
				zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
					return lvl >= opt.level
				}),
			),
		)
	}

	logger := zap.New(core,
		zap.AddCaller(),
		zap.ErrorOutput(stderr),
		// zap.AddCallerSkip(1), // 文件调用层级
	)

	for key, value := range opt.fields {
		logger = logger.WithOptions(zap.Fields(zapcore.Field{Key: key, Type: zapcore.StringType, String: value}))
	}
	logMap.Store(key, logger)
	return logger, nil
}

func Get(key string) *zap.Logger {
	v, ok := logMap.Load(key)
	if !ok {
		return nil
	}
	return v.(*zap.Logger)
}

func CloseAll() {
	logMap.Range(func(_, v interface{}) bool {
		v.(*zap.Logger).Sync()
		return true
	})
}
