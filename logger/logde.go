package logger

import (
	_ "embed"
	"fmt"
	"github.com/getsentry/sentry-go"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"time"
)

const (
	TimeDivision = "time"
	SizeDivision = "size"

	_defaultEncoding             = "console"
	_defaultDivision             = "size"
	_defaultUnit                 = Hour
	_defaultStatFormat           = "2006-01-02 15:00:00"
	_defaultStatClearIntervalDay = 7
)

var (
	_encoderNameToConstructor = map[string]func(zapcore.EncoderConfig) zapcore.Encoder{
		"console": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
			return zapcore.NewConsoleEncoder(encoderConfig)
		},
		"json": func(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
			return zapcore.NewJSONEncoder(encoderConfig)
		},
	}
)

type Log struct {
	L  *zap.Logger
	Op *LogOptions
}

type LogOptions struct {
	// Encoding sets the logger's encoding. Valid values are "json" and
	// "console", as well as any third-party encodings registered via
	// RegisterEncoder.
	Encoding              string             `json:"encoding" yaml:"encoding" toml:"encoding"`
	InfoFilename          string             `json:"info_filename" yaml:"info_filename" toml:"info_filename"`
	ErrorFilename         string             `json:"error_filename" yaml:"error_filename" toml:"error_filename"`
	MaxSize               int                `json:"max_size" yaml:"max_size" toml:"max_size"` // mb 默认100mb
	MaxBackups            int                `json:"max_backups" yaml:"max_backups" toml:"max_backups"`
	MaxAge                int                `json:"max_age" yaml:"max_age" toml:"max_age"` // day 默认不限
	Compress              bool               `json:"compress" yaml:"compress" toml:"compress"`
	Division              string             `json:"division" yaml:"division" toml:"division"`
	LevelSeparate         bool               `json:"level_separate" yaml:"level_separate" toml:"level_separate"`
	TimeUnit              TimeUnit           `json:"time_unit" yaml:"time_unit" toml:"time_unit"`
	Stacktrace            bool               `json:"stacktrace" yaml:"stacktrace" toml:"stacktrace"`
	SentryConfig          SentryLoggerConfig `json:"sentry_config" yaml:"sentry_config" toml:"sentry_config"`
	closeDisplay          int
	caller                bool
	callerSkip            int
	enableQueue           bool
	enableStats           bool
	queueSize             uint // default 100
	infoQueue             *circularFifoQueue
	errorQueue            *circularFifoQueue
	stats                 *stats // 暂时只记录warning以上日志
	statsFormat           string
	statsClearIntervalDay uint8
}

func infoLevel() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.WarnLevel
	})
}

func warnLevel() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})
}

func New() *LogOptions {
	return &LogOptions{
		Division:              _defaultDivision,
		LevelSeparate:         false,
		TimeUnit:              _defaultUnit,
		Encoding:              _defaultEncoding,
		caller:                false,
		statsFormat:           _defaultStatFormat,
		statsClearIntervalDay: _defaultStatClearIntervalDay,
	}
}

func (c *LogOptions) SetDivision(division string) {
	c.Division = division
}

func (c *LogOptions) CloseConsoleDisplay() {
	c.closeDisplay = 1
}

func (c *LogOptions) SetCaller(b bool) {
	c.caller = b
}

func (c *LogOptions) SetCallerSkip(skip int) {
	c.callerSkip = skip
}

func (c *LogOptions) SetTimeUnit(t TimeUnit) {
	c.TimeUnit = t
}

func (c *LogOptions) SetErrorFile(path string) {
	c.LevelSeparate = true
	c.ErrorFilename = path
}

func (c *LogOptions) SetInfoFile(path string) {
	c.InfoFilename = path
}

func (c *LogOptions) SetEncoding(encoding string) {
	c.Encoding = encoding
}

func (c *LogOptions) SetEnableQueue(enableQueue bool) {
	c.enableQueue = enableQueue
}

func (c *LogOptions) SetQueueSize(queueSize uint) {
	c.queueSize = queueSize
	NewCircularFifoQueue(queueSize)
}

func (c *LogOptions) SetEnableStats(enable bool) {
	c.enableStats = enable
}
func (c *LogOptions) SetStatsFormat(format string) {
	c.statsFormat = format
	if c.stats != nil {
		c.stats.Format = format
	}
}

func (c *LogOptions) QueueSize() uint {
	if c.queueSize < 1 {
		return uint(100)
	}
	return c.queueSize
}

func (c *LogOptions) ErrorQueue() *circularFifoQueue {
	return c.errorQueue
}

func (c *LogOptions) InfoQueue() *circularFifoQueue {
	return c.infoQueue
}

func (c *LogOptions) GetStats() *stats {
	return c.stats
}

func (c *LogOptions) SetStatsClearIntervalDay(statsClearIntervalDay uint8) {
	c.statsClearIntervalDay = statsClearIntervalDay
	if c.stats != nil {
		c.stats.DayClearInterval = statsClearIntervalDay
	}
}

// isOutput whether set output file
func (c *LogOptions) isOutput() bool {
	return c.InfoFilename != ""
}

func (c *LogOptions) sizeDivisionWriter(filename string) io.Writer {
	hook := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
		Compress:   c.Compress,
	}
	return hook
}

func (c *LogOptions) timeDivisionWriter(filename string) io.Writer {
	hook, err := rotatelogs.New(
		filename+c.TimeUnit.Format()+".log",
		rotatelogs.WithMaxAge(time.Duration(int64(24*time.Hour)*int64(c.MaxAge))),
		rotatelogs.WithRotationTime(c.TimeUnit.RotationGap()),
	)

	if err != nil {
		panic(err)
	}
	return hook
}

func (c *LogOptions) InitLogger() *Log {
	var (
		logger             *zap.Logger
		infoHook, warnHook io.Writer
		wsInfo             []zapcore.WriteSyncer
		wsWarn             []zapcore.WriteSyncer
	)

	if c.Encoding == "" {
		c.Encoding = _defaultEncoding
	}
	encoder := _encoderNameToConstructor[c.Encoding]

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "file",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if c.closeDisplay == 0 {
		wsInfo = append(wsInfo, zapcore.AddSync(os.Stdout))
		wsWarn = append(wsWarn, zapcore.AddSync(os.Stdout))
	}

	// zapcore WriteSyncer setting
	if c.isOutput() {
		switch c.Division {
		case TimeDivision:
			infoHook = c.timeDivisionWriter(c.InfoFilename)
			if c.LevelSeparate {
				warnHook = c.timeDivisionWriter(c.ErrorFilename)
			}
		case SizeDivision:
			infoHook = c.sizeDivisionWriter(c.InfoFilename)
			if c.LevelSeparate {
				warnHook = c.sizeDivisionWriter(c.ErrorFilename)
			}
		}
		wsInfo = append(wsInfo, zapcore.AddSync(infoHook))
	}

	c.infoQueue = NewCircularFifoQueue(c.QueueSize())
	c.errorQueue = NewCircularFifoQueue(c.QueueSize())
	if c.enableQueue {
		wsInfo = append(wsInfo, zapcore.AddSync(c.infoQueue))
		wsWarn = append(wsWarn, zapcore.AddSync(c.errorQueue))
	}

	c.stats = NewStats()
	c.stats.Format = c.statsFormat
	c.stats.DayClearInterval = c.statsClearIntervalDay
	if c.enableStats {
		wsWarn = append(wsWarn, zapcore.AddSync(c.stats))
	}

	if c.ErrorFilename != "" {
		wsWarn = append(wsWarn, zapcore.AddSync(warnHook))
	}

	opts := make([]zap.Option, 0)
	cos := make([]zapcore.Core, 0)

	if c.LevelSeparate {
		cos = append(
			cos,
			zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsInfo...), infoLevel()),
			zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsWarn...), warnLevel()),
		)
	} else {
		cos = append(
			cos,
			zapcore.NewCore(encoder(encoderConfig), zapcore.NewMultiWriteSyncer(wsInfo...), zap.InfoLevel),
		)
	}

	opts = append(opts, zap.Development())

	if c.Stacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.WarnLevel))
	}

	if c.caller {
		opts = append(opts, zap.AddCaller())
		if c.callerSkip >= 1 {
			opts = append(opts, zap.AddCallerSkip(c.callerSkip))
		}
	}

	logger = zap.New(zapcore.NewTee(cos...), opts...)

	if c.SentryConfig.DSN != "" {
		// sentrycore配置
		cfg := sentryCoreConfig{
			Level:             zap.ErrorLevel,
			Tags:              c.SentryConfig.Tags,
			DisableStacktrace: !c.SentryConfig.AttachStacktrace,
		}
		// 生成sentry客户端
		sentryClient, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:              c.SentryConfig.DSN,
			Debug:            c.SentryConfig.Debug,
			AttachStacktrace: c.SentryConfig.AttachStacktrace,
			Environment:      c.SentryConfig.Environment,
		})
		if err != nil {
			fmt.Println(err)
		}

		sCore := NewSentryCore(cfg, sentryClient)
		logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, sCore)
		}))
	}

	l := &Log{L: logger, Op: c}
	return l
}

func (c *Log) Info(msg string, args ...zap.Field) {
	c.L.Info(msg, args...)
}

func (c *Log) Error(msg string, args ...zap.Field) {
	c.L.Error(msg, args...)
}

func (c *Log) Warn(msg string, args ...zap.Field) {
	c.L.Warn(msg, args...)
}

func (c *Log) Debug(msg string, args ...zap.Field) {
	c.L.Debug(msg, args...)
}

func (c *Log) Fatal(msg string, args ...zap.Field) {
	c.L.Fatal(msg, args...)
}

func (c *Log) Infof(format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Info(logMsg)
}

func (c *Log) Errorf(format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Error(logMsg)
}

func (c *Log) ErrorE(err error, format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Error(logMsg, c.WithError(err))
}

func (c *Log) ErrEf(err error, format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Error(logMsg, c.With("err", err.Error()))
}

func (c *Log) Warnf(format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Warn(logMsg)
}

func (c *Log) Debugf(format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Debug(logMsg)
}

func (c *Log) Fatalf(format string, args ...interface{}) {
	logMsg := fmt.Sprintf(format, args...)
	c.L.Fatal(logMsg)
}

func (c *Log) With(k string, v interface{}) zap.Field {
	return zap.Any(k, v)
}

func (c *Log) WithError(err error) zap.Field {
	return zap.NamedError("error", err)
}

func (c *Log) Close() *Log {
	var l = &Log{
		L:  c.L,
		Op: c.Op,
	}
	l.Op.InitLogger()
	return l
}
