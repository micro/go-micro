package snssqs

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/logger"
)

type sessClientKey struct{}

const (
	defaultMaxMessages             = 1
	defaultVisibilityTimeout       = 3
	defaultWaitSeconds             = 10
	defaultValidateOnPublish       = false
	defaultValidateHeaderOnPublish = false
)

// Amazon Services
type awsServices struct {
	svcSqs    *sqs.SQS
	svcSns    *sns.SNS
	sess      *session.Session
	accountID string
	options   broker.Options
}

// A subscriber (poller) to an SQS queue
type subscriber struct {
	options   broker.SubscribeOptions
	queueName string
	svc       *sqs.SQS
	URL       string
	exit      chan bool
}

// A wrapper around an SQS message published on an SQS queue and delivered via subscriber
type sqsEvent struct {
	sMessage  *sqs.Message
	svc       *sqs.SQS
	m         *broker.Message
	URL       string
	queueName string
	err       error
}

func init() {
	cmd.DefaultBrokers["snssqs"] = NewBroker
}

// run is designed to run as a goroutine and poll SQS for new messages. Note that it's possible to receive
// more than one message from a single poll depending on the options configured for the plugin
func (s *subscriber) run(hdlr broker.Handler) {
	logger.Debugf("SQS subscription started. Queue:%s, URL: %s", s.queueName, s.URL)

	for {
		select {
		case <-s.exit:
			return
		default:
			result, err := s.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            &s.URL,
				MaxNumberOfMessages: s.getMaxMessages(),
				VisibilityTimeout:   s.getVisibilityTimeout(),
				WaitTimeSeconds:     s.getWaitSeconds(),
				AttributeNames: aws.StringSlice([]string{
					"SentTimestamp", // TODO: not currently exposing this to plugin users
				}),
				MessageAttributeNames: aws.StringSlice([]string{
					"All",
				}),
			})

			if err != nil {
				time.Sleep(time.Second)
				logger.Errorf("Error receiving SQS message: %s", err.Error())
				continue
			}

			if len(result.Messages) == 0 {
				time.Sleep(time.Second)
				continue
			}

			for _, sm := range result.Messages {
				s.handleMessage(sm, hdlr)
			}
		}
	}
}

func (s *subscriber) getMaxMessages() *int64 {
	if v := s.options.Context.Value(maxMessagesKey{}); v != nil {
		v2 := v.(int64)
		return aws.Int64(v2)
	}
	return aws.Int64(defaultMaxMessages)
}

func (s *subscriber) getVisibilityTimeout() *int64 {
	if v := s.options.Context.Value(visibilityTimeoutKey{}); v != nil {
		v2 := v.(int64)
		return aws.Int64(v2)
	}
	return aws.Int64(defaultVisibilityTimeout)
}

func (s *subscriber) getWaitSeconds() *int64 {
	if v := s.options.Context.Value(waitTimeSecondsKey{}); v != nil {
		v2 := v.(int64)
		return aws.Int64(v2)
	}
	return aws.Int64(defaultWaitSeconds)
}

func (s *subscriber) handleMessage(msg *sqs.Message, hdlr broker.Handler) {
	logger.Debugf("Received SQS message: %d bytes", len(*msg.Body))
	m := &broker.Message{
		Header: buildMessageHeader(msg.MessageAttributes),
		Body:   []byte(*msg.Body),
	}

	p := &sqsEvent{
		sMessage:  msg,
		m:         m,
		URL:       s.URL,
		queueName: s.queueName,
		svc:       s.svc,
	}

	if p.err = hdlr(p); p.err != nil {
		fmt.Println(p.err)
	}
	if s.options.AutoAck {
		err := p.Ack()
		if err != nil {
			logger.Errorf("Failed auto-acknowledge of message: %s", err.Error())
		}
	}
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.options
}

func (s *subscriber) Topic() string {
	return s.queueName
}

func (s *subscriber) Unsubscribe() error {
	select {
	case <-s.exit:
		return nil
	default:
		close(s.exit)
		return nil
	}
}

func (p *sqsEvent) Error() error {
	return p.err
}

func (p *sqsEvent) Ack() error {
	_, err := p.svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &p.URL,
		ReceiptHandle: p.sMessage.ReceiptHandle,
	})
	return err
}

func (p *sqsEvent) Topic() string {
	return p.queueName
}

func (p *sqsEvent) Message() *broker.Message {
	return p.m
}

func (b *awsServices) Options() broker.Options {
	return b.options
}

// AWS SDK manages the server address internally
func (b *awsServices) Address() string {
	return ""
}

func (b *awsServices) Connect() error {
	if svc := b.getAwsClient(); svc != nil {
		b.sess = svc
		return nil
	}

	b.sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{},
	}))

	sqsConfig := b.getSQSConfig()
	b.svcSqs = sqs.New(b.sess, sqsConfig)

	snsConfig := b.getSNSConfig()
	b.svcSns = sns.New(b.sess, snsConfig)

	stsConfig := b.getSTSConfig()
	svcSts := sts.New(b.sess, stsConfig)

	input := &sts.GetCallerIdentityInput{}

	result, err := svcSts.GetCallerIdentity(input)
	if err != nil {
		return fmt.Errorf("unable to determine AWS AccountId: %s", err.Error())
	}
	b.accountID = *result.Account

	return nil
}

// Disconnect does nothing as there's no live connection to terminate
func (b *awsServices) Disconnect() error {
	return nil
}

// Init initializes a broker and configures an AWS session and SNSSQS struct
func (b *awsServices) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}

	return nil
}

// Publish publishes a message via SNS
func (b *awsServices) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	if getValidateOnPublish(options.Context) {
		if err := ValidateBody(msg); err != nil {
			return err
		}
	}

	if getValidateHeaderOnPublish(options.Context) {
		if err := ValidateHeader(msg, getHeaderWhitelistOnPublish(options.Context)); err != nil {
			return err
		}
	}

	topicArn := arn.ARN{
		Partition: "aws",
		Service:   "sns",
		Region:    *b.sess.Config.Region,
		AccountID: b.accountID,
		Resource:  topic,
	}.String()

	input := &sns.PublishInput{
		Message:  aws.String(string(msg.Body[:])),
		TopicArn: &topicArn,
	}
	input.MessageAttributes = copyMessageHeader(options.Context, msg)

	logger.Debugf("Publishing SNS message to %s, %d bytes", topic, len(msg.Body))
	if _, err := b.svcSns.Publish(input); err != nil {
		return err
	}

	// Broker interfaces don't let us do anything with message ID or sequence number
	return nil
}

// Subscribe subscribes to an SQS queue, starting a goroutine to poll for messages
func (b *awsServices) Subscribe(queueName string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	queueURL, err := b.urlFromQueueName(queueName)
	if err != nil {
		return nil, err
	}

	options := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   queueName,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	subscriber := &subscriber{
		options:   options,
		URL:       queueURL,
		queueName: queueName,
		svc:       b.svcSqs,
		exit:      make(chan bool),
	}
	go subscriber.run(h)

	return subscriber, nil
}

func (b *awsServices) urlFromQueueName(queueName string) (string, error) {
	resultURL, err := b.svcSqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
			return "", fmt.Errorf("unable to find queue %s: %s", queueName, err.Error())
		}
		return "", fmt.Errorf("unable to determine URL for queue %s: %s", queueName, err.Error())
	}
	return *resultURL.QueueUrl, nil
}

// String returns the name of the broker plugin
func (b *awsServices) String() string {
	return "snssqs"
}

func (b *awsServices) getAwsClient() *session.Session {
	raw := b.options.Context.Value(sessClientKey{})
	if raw != nil {
		s := raw.(*session.Session)
		return s
	}
	return nil
}

func (b *awsServices) getSNSConfig() *aws.Config {
	raw := b.options.Context.Value(snsConfigKey{})
	if raw != nil {
		return raw.(*aws.Config)
	}
	return nil
}

func (b *awsServices) getSQSConfig() *aws.Config {
	raw := b.options.Context.Value(sqsConfigKey{})
	if raw != nil {
		return raw.(*aws.Config)
	}
	return nil
}

func (b *awsServices) getSTSConfig() *aws.Config {
	raw := b.options.Context.Value(stsConfigKey{})
	if raw != nil {
		return raw.(*aws.Config)
	}
	return nil
}

// NewBroker creates a new broker with options
func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	return &awsServices{
		options: options,
	}
}

func copyMessageHeader(ctx context.Context, m *broker.Message) (attribs map[string]*sns.MessageAttributeValue) {
	headerWhitelistOnPublish := getHeaderWhitelistOnPublish(ctx)

	attribs = make(map[string]*sns.MessageAttributeValue)
	for k, v := range m.Header {
		if headerWhitelistOnPublish != nil {
			if _, ok := headerWhitelistOnPublish[k]; !ok {
				logger.Debugf("header not whitelisted, removing: %s", k)
				continue
			}
		}

		attribs[k] = &sns.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}
	return attribs
}

func buildMessageHeader(attribs map[string]*sqs.MessageAttributeValue) map[string]string {
	res := make(map[string]string)

	for k, v := range attribs {
		res[k] = *v.StringValue
	}
	return res
}

// ValidateBody Validate message for the lowest requirements of both SNS and SQS
func ValidateBody(msg *broker.Message) error {
	// SNS requirements
	if len(msg.Body) > 256*1024 {
		return fmt.Errorf("message body over 256kB bytes")
	}
	if !utf8.Valid(msg.Body) {
		return fmt.Errorf("message body does not consist solely of UTF-8 characters")
	}

	// SQS Requirements
	// Only accept the following unicode ranges:
	// #x9 | #xA | #xD | #x20 to #xD7FF | #xE000 to #xFFFD | #x10000 to #x10FFFF

	numWorkers := 8
	runeCh := make(chan rune)
	var err error
	waitGroup := sync.WaitGroup{}

	for i := 0; i < numWorkers; i++ {
		waitGroup.Add(1)
		go func(wg *sync.WaitGroup, rCh <-chan rune, err *error) {
			defer wg.Done()
			for r := range rCh {
				if !unicode.In(r, validSqsRunes) {
					*err = fmt.Errorf("message body contains invalid UTF-8 characters for SQS messages")
				}
			}
		}(&waitGroup, runeCh, &err)
	}

	for _, r := range string(msg.Body) {
		if err != nil {
			close(runeCh)
			return err
		}
		runeCh <- r
	}
	close(runeCh)
	waitGroup.Wait()

	return err
}

func ValidateHeader(msg *broker.Message, whitelist map[string]struct{}) error {
	// SNS Requirement
	// can only have a max of 10 headers (converted to attributes) or silently fails
	if len(msg.Header) > 10 && (whitelist == nil || len(whitelist) > 10) {
		totalHeaders := len(msg.Header)
		if whitelist != nil {
			totalHeaders = 0
			for k := range msg.Header {
				if _, ok := whitelist[k]; ok {
					totalHeaders++
				}
			}
		}
		return fmt.Errorf("too many headers %d (max 10)", totalHeaders)
	}

	// SNS Requirement
	// check for allowable characters in header name (A-Z, a-z, 0-9, -, _, .)
	validSNSAttrName := regexp.MustCompile(`(?i)^[A-Z0-9\-_\.]+$`)
	for k := range msg.Header {
		if whitelist != nil {
			if _, ok := whitelist[k]; !ok {
				continue
			}
		}

		if !validSNSAttrName.MatchString(k) {
			return fmt.Errorf("invlaid characters in header key %s", k)
		}
	}

	return nil
}

func getValidateOnPublish(ctx context.Context) bool {
	if ctx == nil {
		return defaultValidateOnPublish
	}
	if v, ok := ctx.Value(validateOnPublishKey{}).(bool); ok && v {
		return true
	}
	// false by default
	return defaultValidateOnPublish
}

func getValidateHeaderOnPublish(ctx context.Context) bool {
	if ctx == nil {
		return defaultValidateHeaderOnPublish
	}
	if v, ok := ctx.Value(validateHeaderOnPublishKey{}).(bool); ok && v {
		return true
	}
	// false by default
	return defaultValidateHeaderOnPublish
}

func getHeaderWhitelistOnPublish(ctx context.Context) map[string]struct{} {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(headerWhitelistOnPublishKey{}).(map[string]struct{}); ok {
		return v
	}
	return nil
}
