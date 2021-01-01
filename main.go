package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/civil"

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
		log.Fatalf("config load failed: %s", err.Error())
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
		log.Fatalf("NewSlackEventStorage failed: %v\n", err)
		return
	}

	defer ses.BigqueryClient.Close()

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
				log.Fatalf("HTTP response write failed with: %s\n", err.Error())
				return
			}
			return
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			log.Printf("event recieved: %s\n", innerEvent.Type)

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				_, _, err := slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
				if err != nil {
					log.Fatalf("AppMentionEvent response failed with: %s\n", err.Error())
					return
				}

			case *slackevents.MessageEvent:
				ctx := context.TODO()
				user, err := slackClient.GetUserInfoContext(ctx, ev.User)
				if err != nil {
					log.Fatalf("GetUserInfoContext failed with: %s\n", err.Error())
					return
				}

				channel, err := slackClient.GetConversationInfoContext(ctx, ev.Channel, true)
				if err != nil {
					log.Fatalf("GetConversationInfoContext failed with: %s\n", err.Error())
					return
				}

				log.Printf("message event: channel=%s, user=%s\n", channel.Name, user.Name)

				if err := ses.BigqueryInserter.Put(ctx, SlackMessageEvent{
					Event:    innerEvent.Type,
					User:     user.Name,
					Channel:  channel.Name,
					DateTime: civil.DateTimeOf(time.Now()),
				}); err != nil {
					log.Fatalf("bigqueryInserter.Put failed with: %s\n", err.Error())
				}

			case *slackevents.ReactionAddedEvent:
				if err := ses.ReactionAddedEvent(ev); err != nil {
					log.Fatalf("ReactionAddedEvent failed with: %s\n", err.Error())
				}
			}
		}
	})
	fmt.Println("[INFO] Server listening")
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", c.StatsbotHost, c.StatsbotPort), nil); err != nil {
		log.Fatalf("http.ListenAndServe failed with: %s\n", err.Error())
		return
	}
}
