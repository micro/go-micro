package windowseventlog

import "github.com/asim/go-micro/v3/logger"

type Log struct {
}

func NewLogger() *Log {
	return &Log{}
}

func (l *Log) Init(options ...logger.Option) error {
	return nil
}

func (l *Log) Options() logger.Options {
	return nil
}

func (l *Log) Fields(fields map[string]interface{}) *Log {
	return nil
}

func (l *Log) Log(level logger.Level, v ...interface{}) {

}

func (l *Log) Logf(level logger.Level, format string, v ...interface{}) {

}

func (l *Log) String() string {
	return "windowseventlog"
}
