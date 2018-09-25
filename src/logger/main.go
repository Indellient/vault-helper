package logger

import (
	log "github.com/sirupsen/logrus"
	"os"
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
			"src": os.Args[0],
		},
	).Debugf(
		format,
		args...,
	)
}

func Infof(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": os.Args[0],
		},
	).Infof(
		format,
		args...,
	)
}

func Warnf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": os.Args[0],
		},
	).Warnf(
		format,
		args...,
	)
}

func Errorf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": os.Args[0],
		},
	).Errorf(
		format,
		args...,
	)
}

func Fatalf(format string, args ...interface{}) {
	log.WithFields(
		log.Fields{
			"src": os.Args[0],
		},
	).Fatalf(
		format,
		args...,
	)
}
