package snssqs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/client"
)

type maxMessagesKey struct{}

// MaxReceiveMessages indicates how many messages a receive operation should pull
// during any single call
func MaxReceiveMessages(max int64) broker.SubscribeOption {
	return setSubscribeOption(maxMessagesKey{}, max)
}

type visibilityTimeoutKey struct{}

// VisibilityTimeout controls how long a message is hidden from other queue consumers
// before being put back. If a consumer does not delete the message, it will be put back
// even if it was "processed"
func VisibilityTimeout(seconds int64) broker.SubscribeOption {
	return setSubscribeOption(visibilityTimeoutKey{}, seconds)
}

type waitTimeSecondsKey struct{}

// WaitTimeSeconds controls the length of long polling for available messages
func WaitTimeSeconds(seconds int64) broker.SubscribeOption {
	return setSubscribeOption(waitTimeSecondsKey{}, seconds)
}

type validateOnPublishKey struct{}

// ValidateOnPublish determines whether to pre-validate messages before they're published
// This has a significant performance impact
func ValidateOnPublish(validate bool) broker.PublishOption {
	return setPublishOption(validateOnPublishKey{}, validate)
}

func ClientValidateOnPublish(validate bool) client.PublishOption {
	return setClientPublishOption(validateOnPublishKey{}, validate)
}

type snsConfigKey struct{}

// SNSConfig add AWS config options to the sns client
func SNSConfig(c *aws.Config) broker.Option {
	return setBrokerOption(snsConfigKey{}, c)
}

type sqsConfigKey struct{}

// SQSConfig add AWS config options to the sqs client
func SQSConfig(c *aws.Config) broker.Option {
	return setBrokerOption(sqsConfigKey{}, c)
}

type stsConfigKey struct{}

// STSConfig add AWS config options to the sts client
func STSConfig(c *aws.Config) broker.Option {
	return setBrokerOption(stsConfigKey{}, c)
}

type validateHeaderOnPublishKey struct{}

// ValidateHeaderOnPublish validate headers before sending to sns
func ValidateHeaderOnPublish(validate bool) broker.PublishOption {
	return setPublishOption(validateHeaderOnPublishKey{}, validate)
}

// ClientValidateHeaderOnPublish validate headers before sending to sns
func ClientValidateHeaderOnPublish(validate bool) client.PublishOption {
	return setClientPublishOption(validateHeaderOnPublishKey{}, validate)
}

type headerWhitelistOnPublishKey struct{}

// HeaderWhitelist validate headers before sending to sns
func HeaderWhitelistOnPublish(whitelist map[string]struct{}) broker.PublishOption {
	return setPublishOption(headerWhitelistOnPublishKey{}, whitelist)
}

// ClientHeaderWhitelist validate headers before sending to sns
func ClientHeaderWhitelistOnPublish(whitelist map[string]struct{}) client.PublishOption {
	return setClientPublishOption(headerWhitelistOnPublishKey{}, whitelist)
}
