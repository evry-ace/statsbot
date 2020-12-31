package main

import (
	"context"

	language "cloud.google.com/go/language/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

type SlackSentiment struct {
	client *language.Client
}

func NewSlacSentiment() (*SlackSentiment, error) {
	ctx := context.Background()

	languageClient, err := language.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &SlackSentiment{
		client: languageClient,
	}, nil
}

func (s SlackSentiment) analyze(text string) (*languagepb.AnalyzeSentimentResponse, error) {
	ctx := context.Background()

	return s.client.AnalyzeSentiment(ctx, &languagepb.AnalyzeSentimentRequest{
		Document: &languagepb.Document{
			Source: &languagepb.Document_Content{
				Content: text,
			},
			Type: languagepb.Document_PLAIN_TEXT,
		},
		EncodingType: languagepb.EncodingType_UTF8,
	})
}
