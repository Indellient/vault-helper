package logger

import (
	log "github.com/sirupsen/logrus"
	"os"
	path "path/filepath"
)

var (
	filename = path.Base(os.Args[0])
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat:  "01/02/2006 15:04:05.000000 -0700",
		FullTimestamp:    true,
		QuoteEmptyFields: true,
	})
}

func SetLoggingLevel(level string) {
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Fatalf("Count not parse --log-level string '%v': %v", level, err)
	}

	log.SetLevel(logLevel)
}

func Debugf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": filename,
		},
	).Debugf(
		format,
		args...,
	)
}

func Infof(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": filename,
		},
	).Infof(
		format,
		args...,
	)
}

func Warnf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": filename,
		},
	).Warnf(
		format,
		args...,
	)
}

func Errorf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": filename,
		},
	).Errorf(
		format,
		args...,
	)
}

func Fatalf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": filename,
		},
	).Fatalf(
		format,
		args...,
	)
}
