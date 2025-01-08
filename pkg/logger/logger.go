package logger

import (
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func Init(dev bool) {

	if dev {
		cfg := zap.NewDevelopmentConfig()
		UpdateLogger(&cfg)
	} else {
		cfg := zap.NewProductionConfig()
		UpdateLogger(&cfg)
	}

}

func UpdateLogger(config *zap.Config) {
	defaultConfig := zap.NewProductionConfig()
	defaultConfig.OutputPaths = []string{"resolvespec.log"}
	if config == nil {
		config = &defaultConfig
	}

	logger, err := config.Build()
	if err != nil {
		log.Print(err)
		return
	}

	Logger = logger.Sugar()
	Info("ResolveSpec Logger initialized")
}

func Info(template string, args ...interface{}) {
	if Logger == nil {
		log.Printf(template, args...)
		return
	}
	Logger.Infow(fmt.Sprintf(template, args...), "process_id", os.Getpid())
}

func Warn(template string, args ...interface{}) {
	if Logger == nil {
		log.Printf(template, args...)
		return
	}
	Logger.Warnw(fmt.Sprintf(template, args...), "process_id", os.Getpid())
}

func Error(template string, args ...interface{}) {
	if Logger == nil {
		log.Printf(template, args...)
		return
	}
	Logger.Errorw(fmt.Sprintf(template, args...), "process_id", os.Getpid())
}

func Debug(template string, args ...interface{}) {
	if Logger == nil {
		log.Printf(template, args...)
		return
	}
	Logger.Debugw(fmt.Sprintf(template, args...), "process_id", os.Getpid())
}
