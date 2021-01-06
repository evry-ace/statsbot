package main

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
)

type SlackMessageEvent struct {
	DateTime civil.DateTime
	Event    string
	User     string
	Channel  string
	Reaction string
}

// Save implements the ValueSaver interface.
func (i *SlackMessageEvent) Save() (map[string]bigquery.Value, string, error) {
	return map[string]bigquery.Value{
		"datetime": i.DateTime,
		"event":    i.Event,
		"user":     i.User,
		"channel":  i.Channel,
		"reaction": i.Reaction,
	}, bigquery.NoDedupeID, nil
}

type SlackEventStorage struct {
	BigqueryClient   *bigquery.Client
	bigqueryInserter *bigquery.Inserter
	slackClient      *slack.Client
	sentimentClient  *SlackSentiment
	config           *Config
}

func NewSlackEventStorage(
	bigqueryProjectID string,
	bigqueryDatasetID string,
	bigqueryTableID string,
	slackClient *slack.Client,
	config *Config) (*SlackEventStorage, error) {

	ctx := context.Background()
	bigqueryClient, err := bigquery.NewClient(ctx, bigqueryProjectID)
	if err != nil {
		return nil, err
	}
	//defer bigqueryClient.Close()

	bigqueryInserter := bigqueryClient.Dataset(bigqueryDatasetID).Table(bigqueryTableID).Inserter()

	sentimentClient, err := NewSlacSentiment()
	if err != nil {
		return nil, err
	}

	return &SlackEventStorage{
		BigqueryClient:   bigqueryClient,
		bigqueryInserter: bigqueryInserter,
		slackClient:      slackClient,
		sentimentClient:  sentimentClient,
		config:           config,
	}, nil
}

func (s SlackEventStorage) MessageEvent(ev *slackevents.MessageEvent) error {
	// Ignore message_changed events, for some reason they are sent when you post
	// to a thread and tick the "send to channel" checkbox
	if ev.SubType == "message_changed" {
		return nil
	}

	ctx := context.Background()

	user, err := s.slackClient.GetUserInfoContext(ctx, ev.User)
	if err != nil {
		return err
	}

	channel, err := s.slackClient.GetConversationInfoContext(ctx, ev.Channel, true)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"channel": channel.Name,
		"sender":  user.Name,
	}).Debug("MessageEvent recieved")

	if err := s.bigqueryInserter.Put(ctx, SlackMessageEvent{
		Event:   ev.Type,
		User:    user.Name,
		Channel: channel.Name,
		// ChanelIsOpen: channel.IsOpen,
		// MessageInTread: ev.ThreadTimeStamp != ""
		DateTime: civil.DateTimeOf(time.Now()),
	}); err != nil {
		return err
	}

	return nil
}

func (s SlackEventStorage) ReactionAddedEvent(ev *slackevents.ReactionAddedEvent) error {
	ctx := context.Background()

	sender, err := s.slackClient.GetUserInfoContext(ctx, ev.User)
	if err != nil {
		return err
	}

	reciever, err := s.slackClient.GetUserInfoContext(ctx, ev.ItemUser)
	if err != nil {
		return err
	}

	channel, err := s.slackClient.GetConversationInfoContext(ctx, ev.Item.Channel, true)
	if err != nil {
		return err
	}

	// Experimental detects sentiment of reaction text
	if s.config.EnableSentiment {
		sentiment, err := s.sentimentClient.analyze(ev.Reaction)

		if err != nil {
			log.WithFields(log.Fields{
				"text":  ev.Reaction,
				"error": err.Error(),
			}).Error("sentimentClient.analyze failed")
		}

		log.WithFields(log.Fields{
			"text":  ev.Reaction,
			"score": sentiment.DocumentSentiment.Score,
		}).Debug("sentimentClient.analyze")

		//if sentiment.DocumentSentiment.Score >= 0 {
		//	fmt.Println("Sentiment: positive")
		//} else {
		//	fmt.Println("Sentiment: negative")
		//}
	}

	log.WithFields(log.Fields{
		"reaction": ev.Reaction,
		"sender":   sender.Name,
		"reciever": reciever.Name,
	}).Debug("reaction recieved")

	// Add event for reaction given
	if err := s.bigqueryInserter.Put(ctx, SlackMessageEvent{
		DateTime: civil.DateTimeOf(time.Now()),
		Event:    ev.Type,
		User:     sender.Name,
		Channel:  channel.Name,
		Reaction: ev.Reaction,
	}); err != nil {
		return err
	}

	// Add event for reaction recieved
	if err := s.bigqueryInserter.Put(ctx, SlackMessageEvent{
		DateTime: civil.DateTimeOf(time.Now()),
		Event:    "reaction_recieved",
		User:     reciever.Name,
		Channel:  channel.Name,
		Reaction: ev.Reaction,
	}); err != nil {
		return err
	}

	return nil
}
