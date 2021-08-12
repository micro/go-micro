package segmentio

import (
	"context"

	"github.com/asim/go-micro/v3/broker"
	"github.com/segmentio/kafka-go"
)

var (
	DefaultReaderConfig = kafka.WriterConfig{}
	DefaultWriterConfig = kafka.ReaderConfig{}
)

type readerConfigKey struct{}
type writerConfigKey struct{}

func ReaderConfig(c kafka.ReaderConfig) broker.Option {
	return setBrokerOption(readerConfigKey{}, c)
}

func WriterConfig(c kafka.WriterConfig) broker.Option {
	return setBrokerOption(writerConfigKey{}, c)
}

type subscribeContextKey struct{}

// SubscribeContext set the context for broker.SubscribeOption
func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type subscribeReaderConfigKey struct{}

func SubscribeReaderConfig(c kafka.ReaderConfig) broker.SubscribeOption {
	return setSubscribeOption(subscribeReaderConfigKey{}, c)
}

type subscribeWriterConfigKey struct{}

func SubscribeWriterConfig(c kafka.WriterConfig) broker.SubscribeOption {
	return setSubscribeOption(subscribeWriterConfigKey{}, c)
}
