package main

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config stores application configurations
type Config struct {
	StatsbotLogLevel        string `envconfig:"SLACKBOT_LOGLEVEL" default:"warn"`
	StatsbotLogFormat       string `envconfig:"SLACKBOT_LOGFORMAT" default:"text"`
	StatsbotPort            string `envconfig:"SLACKBOT_PORT" default:"8080"`
	StatsbotHost            string `envconfig:"SLACKBOT_HOST" default:"127.0.0.1"`
	StatsbotEnableSentiment bool   `envconfig:"SLACKBOT_REACTION_SENTIMENT_ENABLE" default:"false"`
	SlackSigningSecret      string `envconfig:"SLACK_SIGNING_SECRET" required:"true"`
	SlackOAuthAccessToken   string `envconfig:"SLACK_OAUTH_ACCESS_TOKEN" required:"true"`
	BigQueryProjectID       string `envconfig:"BIGQUERY_PROJECT_ID" required:"true"`
	BigQueryDatasetID       string `envconfig:"BIGQUERY_DATASET_ID" required:"true"`
	BigQueryTableID         string `envconfig:"BIGQUERY_TABLE_ID" required:"true"`
}

// ConfigFromEnvironment loads config from env variables and .env file
func ConfigFromEnvironment() (Config, error) {
	// we do not care if there is no .env file.
	_ = godotenv.Overload()

	var s Config
	err := envconfig.Process("", &s)
	if err != nil {
		return s, err
	}

	return s, nil
}
