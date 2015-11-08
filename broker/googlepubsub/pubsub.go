package googlepubsub

import (
	"errors"
	"io/ioutil"

	"github.com/piemapping/go-micro/broker"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/pubsub"

	"golang.org/x/net/context"
)

var (
	// ErrGooglePubSubNotInitialised indicates that the google pub sub broker was not initialised
	ErrGooglePubSubNotInitialised = errors.New("PubSub not initialised")
)

const (
	headerMsgID         = "_pubsub_msg_id"
	headerMsgAckID      = "_pubsub_msg_ack_id"
	defaultMsgBatchSize = 50
)

type pubsubBroker struct {
	jsonFilePath string
	projectID    string
	ctx          context.Context
}

// NewBroker instantiates a broker that connects to Google Cloud PubSub (https://cloud.google.com/pubsub/)
func NewBroker(authFilePath, projectID string) broker.Broker {
	return &pubsubBroker{
		jsonFilePath: authFilePath,
		projectID:    projectID,
	}
}

type pubsubHandler struct {
	ctx              context.Context
	subscriptionName string
	handlerFunc      func(*broker.Message) error
}

func (psh *pubsubHandler) Ack(msg *broker.Message) error {
	msgAckID := getMsgAckID(msg)

	return pubsub.Ack(psh.ctx, psh.subscriptionName, msgAckID)
}

func (psh *pubsubHandler) Handle(msg *broker.Message) error {
	if psh.handlerFunc == nil {
		return nil
	}

	return psh.handlerFunc(msg)
}

func getMsgID(msg *broker.Message) string {
	return msg.Header[headerMsgID]
}

func getMsgAckID(msg *broker.Message) string {
	return msg.Header[headerMsgAckID]
}

func pubsubMsgToBrokerMsg(psMsg *pubsub.Message) *broker.Message {
	bmsg := &broker.Message{
		Body:   psMsg.Data,
		Header: psMsg.Attributes,
	}

	if bmsg.Header == nil {
		bmsg.Header = make(map[string]string)
	}

	bmsg.Header[headerMsgAckID] = psMsg.AckID
	bmsg.Header[headerMsgID] = psMsg.ID

	return bmsg
}

func (ps *pubsubBroker) Address() string {
	return ""
}

func (ps *pubsubBroker) getContext() (context.Context, error) {
	// if no JSON file path is specified, use the default authentication mechanism
	// as we assume we are running inside Google Cloud Engine
	if len(ps.jsonFilePath) == 0 {
		ctx := context.TODO()
		ts, err := google.DefaultTokenSource(ctx,
			pubsub.ScopeCloudPlatform,
			pubsub.ScopePubSub,
		)

		if err != nil {
			return nil, err
		}

		return cloud.NewContext(ps.projectID, oauth2.NewClient(ctx, ts)), nil
	}
	// Otherwise... just use the JSON file key
	jsonKey, err := ioutil.ReadFile(ps.jsonFilePath)
	if err != nil {
		return nil, err
	}

	// if there is no json config key assume we are using the default credentials
	conf, err := google.JWTConfigFromJSON(
		jsonKey,
		pubsub.ScopeCloudPlatform,
		pubsub.ScopePubSub,
	)

	if err != nil {
		return nil, err
	}
	return cloud.NewContext(ps.projectID, conf.Client(oauth2.NoContext)), nil

}

func (ps *pubsubBroker) Init() error {
	ctx, err := ps.getContext()
	if err != nil {
		return err
	}

	ps.ctx = ctx

	return nil
}

func (ps *pubsubBroker) Connect() error {
	// We do not need to do anything at this stage
	return nil
}

func (ps *pubsubBroker) Disconnect() error {
	// We do not need to do anything at this stage either
	return nil
}

func (ps *pubsubBroker) Publish(topic string, msg *broker.Message) error {
	if ps.ctx == nil {
		return ErrGooglePubSubNotInitialised
	}

	psMsg := &pubsub.Message{
		Data: msg.Body,
		// Attributes map more to metadata and not only headers. But this will do just fine in the meantime
		Attributes: msg.Header,
	}

	if err := ps.createTopicIfNotExists(topic); err != nil {
		return err
	}

	// We are not interested in the returned message IDs upon success
	_, err := pubsub.Publish(ps.ctx, topic, psMsg)

	return err
}

func (ps *pubsubBroker) NewSubscriber(name, topic string) (broker.Subscriber, error) {
	subscriber := &pullsub{
		ctx:       ps.ctx,
		name:      name,
		topic:     topic,
		batchSize: defaultMsgBatchSize,
		stopped:   make(chan bool),
	}

	return subscriber, nil
}

func (ps *pubsubBroker) createTopicIfNotExists(topic string) error {
	exists, err := pubsub.TopicExists(ps.ctx, topic)
	if err != nil {
		return err
	}

	if !exists {
		return pubsub.CreateTopic(ps.ctx, topic)
	}

	return nil
}
