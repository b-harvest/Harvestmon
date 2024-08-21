package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	database "github.com/b-harvest/Harvestmon/database"
	"github.com/b-harvest/Harvestmon/log"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/rs/zerolog"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// You more than likely want your "Bot User OAuth Access Token" which starts with "xoxb-"
var (
	api               *slack.Client
	signingSecret     string
	verificationToken string
	db                *gorm.DB
	CommitID          string
)

const MARKER_FROM = "slack"

func init() {
	api = slack.New(os.Getenv("TOKEN"))
	signingSecret = os.Getenv("SLACK_SIGNING_SECRET")
	CommitID = os.Getenv("COMMIT_ID")
	verificationToken = os.Getenv("VERIFICATION_TOKEN")

	sqlDB, err := database.GetDatabase("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	db, err = gorm.Open(gorm_mysql.New(gorm_mysql.Config{Conn: sqlDB}))
	if err != nil {
		log.Fatal(err)
	}
	logLevelDebug := flag.Bool("debug", false, "allow showing debug log")

	flag.Parse()

	if *logLevelDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

}

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	req, err := http.NewRequest(event.HTTPMethod, event.Path, bytes.NewReader([]byte(event.Body)))
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}

	for key, value := range event.Headers {
		req.Header.Add(key, value)
	}

	rr := &ResponseRecorder{
		HeaderMap: make(map[string]string),
		Body:      new(bytes.Buffer),
	}

	handleAction(rr, req)

	log.Debug("Complete handling.... ")

	if rr.StatusCode == 0 {
		rr.StatusCode = http.StatusOK
		log.Debug("StatusCode set as 200 ok to prevent retrying")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: rr.StatusCode,
		Body:       rr.Body.String(),
		Headers:    rr.HeaderMap,
	}, nil
}

type ResponseRecorder struct {
	StatusCode int
	HeaderMap  map[string]string
	Body       *bytes.Buffer
}

func (rr *ResponseRecorder) Header() http.Header {
	return http.Header{}
}

func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	return rr.Body.Write(data)
}

func (rr *ResponseRecorder) WriteHeader(statusCode int) {
	rr.StatusCode = statusCode
}

// AddHeader is a helper method to convert http.Header to map[string]string
func (rr *ResponseRecorder) AddHeader(key, value string) {
	rr.HeaderMap[key] = value
}

func main() {
	lambda.Start(handler)
}

//func main() {
//
//	http.HandleFunc("/events-endpoint", handleAction)
//	fmt.Println("[INFO] Server listening")
//	http.ListenAndServe(":8888", nil)
//}

const (
	// action is used for slack attament action.
	actionSelect = "select"
	actionCancel = "cancel"

	callStop  = "stop"
	callStart = "start"

	agentStopCallback  = "agent_stop"
	agentStartCallback = "agent_start"

	checkEmoticon  = ":white_check_mark:"
	yellowEmoticon = ":large_yellow_circle:"
	failedEmoticon = ":x:"
)

var actions = map[string]func(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, agentName, stopTime string){
	callStop:  stopAction,
	callStart: startAction,
}

var selectActions = map[string]func(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, stopTime string){
	callStop:  selectStopAction,
	callStart: selectStartAction,
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	var (
		body []byte
		err  error
	)

	if r.Header.Get("Content-Type") != "application/json" {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Extract the payload
		payload := r.PostFormValue("payload")

		// Decode the URL-encoded payload
		decodedPayload, err := url.QueryUnescape(payload)
		if err != nil {
			http.Error(w, "Failed to decode payload", http.StatusBadRequest)
			return
		}
		r.Header.Set("Content-Type", "application/json")
		body = []byte(decodedPayload)
	} else {
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			log.Debug(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	log.Debug(string(body))

	sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
	if err != nil {
		log.Debug(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := sv.Ensure(); err != nil {
		if eventsAPIEvent.Token != verificationToken {
			w.WriteHeader(http.StatusUnauthorized)
			log.Debug(err)
			return
		}
	}

	baseRepository := repository.BaseRepository{
		DB:       *db,
		CommitId: CommitID,
	}

	agentMarkRepository := repository.AgentMarkRepository{
		BaseRepository: baseRepository,
	}

	agentRepository := repository.AgentRepository{
		BaseRepository: baseRepository,
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		var r *slackevents.ChallengeResponse
		err = json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		challenge := map[string]string{
			"challenge": r.Challenge,
		}
		cBytes, _ := json.Marshal(challenge)
		log.Debug(botFormatf("URLVerification - challenge: %s", string(cBytes)))
		_, err = w.Write(cBytes)
		if err != nil {
			log.Error(err)
		}
		break
	case slackevents.CallbackEvent:
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// @bot [action] [agent-name] [duration?]

			metion := regexp.MustCompile(`<@[A-Z0-9]+>`).FindString(ev.Text)
			afterMentionText := strings.TrimPrefix(ev.Text, ev.Text[:strings.Index(ev.Text, metion)+len(metion)+1])

			log.Debug(botFormatf(afterMentionText))
			params := strings.Split(afterMentionText, " ")

			paramsLen := len(params)
			switch paramsLen {
			case 1:
				if action, exists := selectActions[params[0]]; exists {
					action(ev, w, agentRepository, agentMarkRepository, "")
					return
				}
				break
			case 2:
				if action, exists := selectActions[params[0]]; exists {
					action(ev, w, agentRepository, agentMarkRepository, params[1])
					return
				}
				break
			case 3:
				if action, exists := actions[params[0]]; exists {
					action(ev, w, agentRepository, agentMarkRepository, params[1], params[2])
					return
				}
				break
			}
			_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("Invalid params. The command must conform to the following format.\n- @[bot] [action] \n-@@[bot] [action] [duration] \n- @@[bot] [action] [duration] [agent-name]"), false))
		}
		break
	case string(slack.InteractionTypeInteractionMessage):
		var interactionCallback slack.InteractionCallback
		err = json.Unmarshal(body, &interactionCallback)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		callbackAction := interactionCallback.ActionCallback.AttachmentActions[0]
		log.Debug(botFormatf("callbackAction: %s", callbackAction.Name))
		now := time.Now()
		agentMarks, err := agentMarkRepository.FindAgentMarkByAgentNameAndTime(interactionCallback.OriginalMessage.ThreadTimestamp, now)
		if err != nil || len(agentMarks) == 0 {
			msg := fmt.Sprintf("%s action already completed.", failedEmoticon)
			_, _, err = api.PostMessage(interactionCallback.Channel.ID, slack.MsgOptionText(msg, false), slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
			return
		}
		agentMark := agentMarks[0]

		if agentMarks[0].AgentName != "" && agentMarks[0].MarkStart != nil && agentMarks[0].MarkerUserIdentity != "" {
			err = agentMarkRepository.Delete(agentMark)
		} else {
			_, _, err = api.PostMessage(interactionCallback.Channel.ID, slack.MsgOptionText("task already done", false), slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
		}
		if err != nil {
			_, _, err = api.PostMessage(interactionCallback.Channel.ID, slack.MsgOptionText(fmt.Sprintf("%s %v", failedEmoticon, err.Error()), false), slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
			return
		}

		switch callbackAction.Name {
		case actionSelect:
			switch interactionCallback.CallbackID {
			case agentStopCallback:
				agentMark.AgentName = callbackAction.SelectedOptions[0].Value

				err = agentMarkRepository.Save(agentMark)
				if err != nil {
					_, _, err = api.PostMessage(interactionCallback.Channel.ID, slack.MsgOptionText(fmt.Sprintf("%s failed to save agentMark: %s", failedEmoticon, err.Error()), false), slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
					return
				}

				msg := fmt.Sprintf("%s <@%s> \nDisabled alert - *%s* \nuntil %v UTC\n(%v) left...", checkEmoticon, agentMark.MarkerUserIdentity, agentMark.AgentName, agentMark.MarkEnd.Format("2006-01-02 15:04:05"), agentMark.MarkEnd.Sub(now).Round(1*time.Minute))

				_, _, err = api.PostMessage(interactionCallback.Channel.ID,
					slack.MsgOptionText(msg, false),
					slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
				return
			case agentStartCallback:
				agentName := callbackAction.SelectedOptions[0].Value

				var until = new(time.Time)
				if agentMark.MarkEnd != nil {
					until = agentMark.MarkEnd
				} else {
					*until = time.Now()
				}

				var realAgentMarks []repository.AgentMark

				if realAgentMarks, err = agentMarkRepository.FindAgentMarkByAgentNameAndTime(agentName, now); err != nil || len(realAgentMarks) == 0 {
					msg := fmt.Sprintf("%s <@%s>\nAlert already started - *%s*", yellowEmoticon, agentMark.MarkerUserIdentity, agentName)

					_, _, err = api.PostMessage(interactionCallback.Channel.ID,
						slack.MsgOptionText(msg, false),
						slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
					return
				}

				for _, realAgentMark := range realAgentMarks {
					realAgentMark.MarkEnd = until
					err = agentMarkRepository.Save(realAgentMark)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
				msg := fmt.Sprintf("%s <@%s> \nAlert successfully started  - *%s*\nIt'll start after %s UTC", checkEmoticon, agentMark.MarkerUserIdentity, agentName, until.Format("2006-01-02 15:04:05"))

				_, _, err = api.PostMessage(interactionCallback.Channel.ID,
					slack.MsgOptionText(msg, false),
					slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
				break
			default:
				log.Error(errors.New("cannot parse callbackId"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case actionCancel:
			title := fmt.Sprintf("%s Canceled the request", failedEmoticon)
			_, _, err = api.PostMessage(interactionCallback.Channel.ID, slack.MsgOptionText(title, false), slack.MsgOptionTS(interactionCallback.OriginalMessage.ThreadTimestamp))
			return
		}
	}

}

func stopAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, agentName, stopTime string) {
	var until time.Time

	if agent, err := agentRepository.FindAgentByAgentName(agentName); err == nil && agent != nil {
		if until, err = time.Parse("2006-01-02 15:04:05", stopTime); err == nil {
		} else if duration, err := time.ParseDuration(stopTime); err == nil {
			until = time.Now().Add(duration)
		} else {
			until = time.Now().Add(30 * time.Minute)
		}

		now := time.Now()
		agentMark := repository.AgentMark{
			AgentName:          agent.AgentName,
			MarkStart:          &now,
			MarkEnd:            &until,
			MarkerUserIdentity: ev.User,
			MarkerFrom:         MARKER_FROM,
		}
		err = agentMarkRepository.Save(agentMark)

		if err != nil {
			msg := fmt.Sprintf("Error occurred while disabling alert %v", err)
			_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
			if err != nil {
				log.Error(err)
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		msg := fmt.Sprintf("%s <@%s> \nDisabled alert - *%s* \nuntil %v UTC\n(%v) left...", checkEmoticon, agentMark.MarkerUserIdentity, agentMark.AgentName, agentMark.MarkEnd.Format("2006-01-02 15:04:05"), agentMark.MarkEnd.Sub(now).Round(1*time.Minute))
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		if err != nil {
			log.Error(err)
		}
		return
	} else {
		msg := fmt.Sprintf("%s Didn't find any agent with received name %s", failedEmoticon, agentName)
		log.Warn(msg)

		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		if err != nil {
			log.Error(err)
		}
		return
	}
}

func startAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, agentName, _ string) {

	if agent, err := agentRepository.FindAgentByAgentName(agentName); err != nil && agent != nil {
		agentMarks, err := agentMarkRepository.FindAgentMarkByAgentNameAndTime(agentName, time.Now())
		if err != nil {
			msg := fmt.Sprintf("%s Error occurred while finding agent mark %v", failedEmoticon, err)
			_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
			if err != nil {
				log.Error(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if len(agentMarks) == 0 {
			msg := fmt.Sprintf("%s <@%s>\nAlert already started - *%s*", yellowEmoticon, ev.User, agentName)
			_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
			if err != nil {
				log.Error(err)
			}
			return
		}

		now := time.Now()

		for _, agentMark := range agentMarks {
			agentMark.MarkEnd = &now
			err = agentMarkRepository.Save(agentMark)

			if err != nil {
				msg := fmt.Sprintf("%s %v", failedEmoticon, err)

				_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
				if err != nil {
					log.Error(err)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		msg := fmt.Sprintf("%s <@%s> \nAlert successfully started  - *%s*\nIt'll start after %s UTC", checkEmoticon, ev.User, agentName, now.Format("2006-01-02 15:04:05"))

		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		if err != nil {
			log.Error(err)
		}
		return
	} else {
		msg := fmt.Sprintf("%s Didn't find any agent with received name %s", failedEmoticon, agentName)

		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		if err != nil {
			log.Error(err)
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func selectStopAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, stopTime string) {
	var (
		until time.Time
		err   error
	)

	if until, err = time.Parse("2006-01-02 15:04:05", stopTime); err == nil {
	} else if duration, err := time.ParseDuration(stopTime); err == nil {
		until = time.Now().Add(duration)
	} else {
		until = time.Now().Add(30 * time.Minute)
	}
	log.Debug(until)

	agents, err := agentRepository.FindAll()
	if err != nil {
		log.Error(err)
	}

	var attachmentActionOptions []slack.AttachmentActionOption

	for _, agent := range agents {
		attachmentActionOptions = append(attachmentActionOptions, slack.AttachmentActionOption{
			Text:  agent.AgentName,
			Value: agent.AgentName,
		})
	}

	attachment := slack.Attachment{
		Text:       fmt.Sprintf("Which agent do you want to stop? :bharvest: "),
		Color:      "#3AA3E3",
		CallbackID: agentStopCallback,
		Actions: []slack.AttachmentAction{
			{
				Name:    actionSelect,
				Type:    "select",
				Options: attachmentActionOptions,
			},
			{
				Name:  actionCancel,
				Text:  "Cancel",
				Type:  "button",
				Style: "danger",
			},
		},
	}

	now := time.Now()

	timestamp := ev.ThreadTimeStamp
	if timestamp == "" {
		timestamp = ev.TimeStamp
	}
	agentMark := repository.AgentMark{
		AgentName:          timestamp,
		MarkStart:          &now,
		MarkEnd:            &until,
		MarkerUserIdentity: ev.User,
		MarkerFrom:         MARKER_FROM,
	}

	err = agentMarkRepository.Save(agentMark)
	if err != nil {
		log.Error(err)
	}

	_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionTS(ev.TimeStamp), slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Error(err)
	}
}

func selectStartAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentMarkRepository repository.AgentMarkRepository, stopTime string) {
	var (
		until = new(time.Time)
		err   error
	)

	if *until, err = time.Parse("2006-01-02 15:04:05", stopTime); err == nil {
	} else if duration, err := time.ParseDuration(stopTime); err == nil {
		*until = time.Now().Add(duration)
	} else {
		until = nil
	}
	log.Debug(until)

	agents, err := agentRepository.FindAll()
	if err != nil {
		log.Error(err)
	}

	var attachmentActionOptions []slack.AttachmentActionOption

	for _, agent := range agents {
		attachmentActionOptions = append(attachmentActionOptions, slack.AttachmentActionOption{
			Text:  agent.AgentName,
			Value: agent.AgentName,
		})
	}

	attachment := slack.Attachment{
		Text:       fmt.Sprintf("Which agent do you want to start? :bharvest: "),
		Color:      "#3AA3E3",
		CallbackID: agentStartCallback,
		Actions: []slack.AttachmentAction{
			{
				Name:    actionSelect,
				Type:    "select",
				Options: attachmentActionOptions,
			},
			{
				Name:  actionCancel,
				Text:  "Cancel",
				Type:  "button",
				Style: "danger",
			},
		},
	}

	now := time.Now()

	timestamp := ev.ThreadTimeStamp
	if timestamp == "" {
		timestamp = ev.TimeStamp
	}
	agentMark := repository.AgentMark{
		AgentName:          timestamp,
		MarkStart:          &now,
		MarkEnd:            until,
		MarkerUserIdentity: ev.User,
		MarkerFrom:         MARKER_FROM,
	}

	err = agentMarkRepository.Save(agentMark)
	if err != nil {
		log.Error(err)
	}

	_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionTS(ev.TimeStamp), slack.MsgOptionAttachments(attachment))
	if err != nil {
		log.Error(err)
	}
}

func botFormatf(msg string, args ...any) string {
	return fmt.Sprintf("[slack-bot] "+msg, args...)
}
