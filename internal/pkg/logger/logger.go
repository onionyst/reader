package logger

import (
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	writerhook "github.com/sirupsen/logrus/hooks/writer"

	"reader/internal/pkg/utils"
)

func New() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)

	l.SetFormatter(&tzFormatter{
		Formatter: &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.DateTime,
			DisableColors:   true,
		},
		loc: utils.Beijing,
	})

	l.SetOutput(io.Discard)
	l.AddHook(&writerhook.Hook{
		Writer:    os.Stdout,
		LogLevels: []logrus.Level{logrus.InfoLevel, logrus.DebugLevel, logrus.TraceLevel, logrus.WarnLevel},
	})
	l.AddHook(&writerhook.Hook{
		Writer:    os.Stderr,
		LogLevels: []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel},
	})

	return l
}

type tzFormatter struct {
	logrus.Formatter
	loc *time.Location
}

func (t *tzFormatter) Format(e *logrus.Entry) ([]byte, error) {
	ne := *e
	ne.Time = e.Time.In(t.loc)
	return t.Formatter.Format(&ne)
}
