package main

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config stores application configurations
type Config struct {
	LogLevel              string `envconfig:"SLACKBOT_LOGLEVEL" default:"warn"`
	LogFormat             string `envconfig:"SLACKBOT_LOGFORMAT" default:"text"`
	Port                  string `envconfig:"SLACKBOT_PORT" default:"8080"`
	Host                  string `envconfig:"SLACKBOT_HOST" default:"0.0.0.0"`
	EnableSentiment       bool   `envconfig:"SLACKBOT_REACTION_SENTIMENT_ENABLE" default:"false"`
	SlackSigningSecret    string `envconfig:"SLACK_SIGNING_SECRET" required:"true"`
	SlackOAuthAccessToken string `envconfig:"SLACK_OAUTH_ACCESS_TOKEN" required:"true"`
	BigQueryProjectID     string `envconfig:"BIGQUERY_PROJECT_ID" required:"true"`
	BigQueryDatasetID     string `envconfig:"BIGQUERY_DATASET_ID" required:"true"`
	BigQueryTableID       string `envconfig:"BIGQUERY_TABLE_ID" required:"true"`
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
