package xlog

import (
	"fmt"
	"os"
	rsync "sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	timeKey         = "time"
	EncodingJson    = "json"
	EncodingConsole = "console"
	FileMode        = "file"
)

var (
	levels = map[string]zapcore.Level{
		"debug": zap.DebugLevel,
		"info":  zap.InfoLevel,
		"error": zap.ErrorLevel,
		"warn":  zap.WarnLevel,
		"panic": zap.PanicLevel,
		"fatal": zap.FatalLevel,
	}

	atomicConf   *XLogConf
	atomicLogger *XLog
	mutex        rsync.RWMutex
)

type (
	XLogConf struct {
		ServiceName string
		// log path
		Path string
		// log file name
		Filename string
		//	file or stdout
		Mode string
		//	json or console
		Encoding   string
		TimeFormat string
		//	debug, info, error, warn, panic, fatal
		Level    string
		Compress bool
		KeepDays int
		MaxSize  int
	}
	XLog struct {
		conf *XLogConf

		instance *zap.Logger
	}
)

func init() {
	if atomicConf == nil {
		atomicConf = &XLogConf{}
		defaultConf(atomicConf)
	}
	atomicLogger = &XLog{
		instance: instance(*atomicConf),
	}
}

func Load(conf *XLogConf) {
	mutex.Lock()
	defer mutex.Unlock()

	defaultConf(conf)
	atomicConf = conf
	atomicLogger = &XLog{conf: conf}
	atomicLogger.instance = instance(*conf)
}

func Write() *zap.Logger {
	mutex.RLock()
	defer mutex.RUnlock()

	return atomicLogger.instance
}

func instance(conf XLogConf) *zap.Logger {
	opts := []zap.Option{
		zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel),
	}
	if len(conf.ServiceName) > 0 {
		opts = append(opts, zap.Fields(zap.String("service", conf.ServiceName)))
	}

	var write zapcore.WriteSyncer
	switch conf.Mode {
	case FileMode:
		write = sync(conf)
	default:
		write = zapcore.Lock(os.Stdout)
	}

	// logs level default : debug
	level, ok := levels[conf.Level]
	if !ok {
		level = zap.DebugLevel
	}
	return zap.New(zapcore.NewCore(encoder(conf), write, level), opts...)
}

func sync(conf XLogConf) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename: fmt.Sprintf("%s/%s", conf.Path, conf.Filename),
		Compress: conf.Compress,
		MaxAge:   conf.KeepDays,
		MaxSize:  conf.MaxSize,
	})
}

func encoder(conf XLogConf) zapcore.Encoder {
	var encoder zapcore.Encoder
	econf := zap.NewProductionEncoderConfig()
	econf.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(conf.TimeFormat))
	}
	if conf.Level == "debug" {
		econf.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	} else {
		econf.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	econf.TimeKey = timeKey
	switch conf.Encoding {
	case EncodingJson:
		encoder = zapcore.NewJSONEncoder(econf)
	default:
		encoder = zapcore.NewConsoleEncoder(econf)
	}
	return encoder
}

func defaultConf(conf *XLogConf) {
	if len(conf.Path) == 0 {
		path, _ := os.Getwd()
		conf.Path = fmt.Sprintf("%s/logs", path)
	}

	if len(conf.Level) == 0 {
		conf.Level = "debug"
	}

	if len(conf.Filename) == 0 {
		conf.Filename = "app.log"
	}

	if len(conf.Encoding) == 0 {
		conf.Encoding = "console"
	}

	if len(conf.TimeFormat) == 0 {
		conf.TimeFormat = "2006-01-02 15:04:05"
	}

	if !conf.Compress {
		conf.Compress = true
	}
}
