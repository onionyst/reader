package logging

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	writerhook "github.com/sirupsen/logrus/hooks/writer"
	"golang.org/x/term"

	"reader/internal/pkg/utils"
)

func LogError(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			logrus.Error(c.Errors)
		}
	}
}

type TZFormatter struct {
	logrus.Formatter
	loc *time.Location
}

func (t *TZFormatter) Format(e *logrus.Entry) ([]byte, error) {
	ne := *e
	ne.Time = e.Time.In(t.loc)
	return t.Formatter.Format(&ne)
}

func Setup() *logrus.Logger {
	noColor := !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stderr.Fd())) || os.Getenv("NO_COLOR") != ""

	gin.DefaultWriter = os.Stdout
	gin.DefaultErrorWriter = os.Stderr
	if noColor {
		gin.DisableConsoleColor()
	}

	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(&TZFormatter{
		Formatter: &logrus.TextFormatter{
			ForceColors:     !noColor,
			FullTimestamp:   true,
			TimestampFormat: time.DateTime,
			DisableColors:   noColor,
		},
		loc: utils.Beijing,
	})
	l.AddHook(&writerhook.Hook{
		Writer:    os.Stdout,
		LogLevels: []logrus.Level{logrus.InfoLevel, logrus.DebugLevel, logrus.TraceLevel, logrus.WarnLevel},
	})
	l.AddHook(&writerhook.Hook{
		Writer:    os.Stderr,
		LogLevels: []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel},
	})
	l.SetLevel(logrus.InfoLevel)

	return l
}
