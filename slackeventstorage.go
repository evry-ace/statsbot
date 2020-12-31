package main

import (
	"context"
	"fmt"
	"log"
	"time"

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
	BigqueryInserter *bigquery.Inserter
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
		BigqueryInserter: bigqueryInserter,
		slackClient:      slackClient,
		sentimentClient:  sentimentClient,
		config:           config,
	}, nil
}

func (s SlackEventStorage) MessageEvent(m SlackMessageEvent) error {
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

	// Detects the sentiment of the text.
	sentiment, err := s.sentimentClient.analyze(ev.Reaction)
	if err != nil {
		log.Fatalf("Failed to analyze text: %v", err)
	}

	fmt.Printf("Sentiment score of '%v' is %f\n", ev.Reaction, sentiment.DocumentSentiment.Score)
	//if sentiment.DocumentSentiment.Score >= 0 {
	//	fmt.Println("Sentiment: positive")
	//} else {
	//	fmt.Println("Sentiment: negative")
	//}

	fmt.Printf(
		"Recieved '%s' given to %s from %s in %s\n",
		ev.Reaction,
		reciever.Name,
		sender.Name,
		channel.Name,
	)

	// Add event for reaction given
	if err := s.BigqueryInserter.Put(ctx, SlackMessageEvent{
		DateTime: civil.DateTimeOf(time.Now()),
		Event:    ev.Type,
		User:     sender.Name,
		Channel:  channel.Name,
		Reaction: ev.Reaction,
	}); err != nil {
		return err
	}

	// Add event for reaction recieved
	if err := s.BigqueryInserter.Put(ctx, SlackMessageEvent{
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
