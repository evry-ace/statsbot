package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func init() {
	setupLogging()
}

func main() {
	// Load Config
	c, err := ConfigFromEnvironment()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("config load failed")
		return
	}

	// Slack API
	slackClient := slack.New(c.SlackOAuthAccessToken)

	// SlackEventStorage
	ses, err := NewSlackEventStorage(
		c.BigQueryProjectID,
		c.BigQueryDatasetID,
		c.BigQueryTableID,
		slackClient,
		&c,
	)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("NewSlackEventStorage failed")
		return
	}

	defer ses.BigqueryClient.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/events-endpoint", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sv, err := slack.NewSecretsVerifier(r.Header, c.SlackSigningSecret)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			if err := json.Unmarshal([]byte(body), &r); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			if _, err := w.Write([]byte(r.Challenge)); err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("HTTP response w.write failed")
				return
			}
			return
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			log.WithFields(log.Fields{
				"event": innerEvent.Type,
			}).Info("event recieved")

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				_, _, err := slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("AppMentionEvent response failed")
				}

			case *slackevents.MessageEvent:
				log.WithFields(log.Fields{
					"event": fmt.Sprintf("%+v", ev),
				}).Trace("slackevents.MessageEvent recieved")

				if err := ses.MessageEvent(ev); err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("ses.MessageEvent failed")
				}

			case *slackevents.ReactionAddedEvent:
				log.WithFields(log.Fields{
					"event": fmt.Sprintf("%+v", ev),
				}).Trace("slackevents.ReactionAddedEvent recieved")

				if err := ses.ReactionAddedEvent(ev); err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("ses.ReactionAddedEvent failed")
				}
			}
		}
	})

	log.WithFields(log.Fields{
		"host": c.Host,
		"port": c.Port,
	}).Info("starting server")

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", c.Host, c.Port), nil); err != nil {
		log.WithFields(log.Fields{
			"host":  c.Host,
			"port":  c.Port,
			"error": err.Error(),
		}).Fatal("http.ListenAndServe failed")
	}
}
