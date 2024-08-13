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
	"gopkg.in/yaml.v3"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// You more than likely want your "Bot User OAuth Access Token" which starts with "xoxb-"
var (
	api           = slack.New(os.Getenv("TOKEN"))
	signingSecret string
	db            *gorm.DB
	CommitID      string
)

func init() {
	signingSecret = os.Getenv("SLACK_SIGNING_SECRET")
	CommitID = os.Getenv("COMMIT_ID")

	configBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	dbConfig := new(database.Database)
	err = yaml.Unmarshal(configBytes, &dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	if dbConfig.User == "" {
		dbConfig.User = os.Getenv("DB_USER")
	}
	if dbConfig.Password == "" {
		dbConfig.Password = os.Getenv("DB_PASSWORD")
	}
	if dbConfig.Host == "" {
		dbConfig.Host = os.Getenv("DB_HOST")
	}
	if dbConfig.Port == 0 {
		port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
		dbConfig.Port = port
	}
	if dbConfig.DbName == "" {
		dbConfig.DbName = os.Getenv("DB_NAME")
	}
	if dbConfig.AwsRegion == "" {
		dbConfig.DbName = os.Getenv("DB_AWS_REGION")
	}

	sqlDB := database.GetDatabase(dbConfig)

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
		req.Header.Set(key, value)
	}

	rr := &ResponseRecorder{
		HeaderMap: make(map[string]string),
		Body:      new(bytes.Buffer),
	}

	handleAction(rr, req)

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

var actions = map[string]func(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentName, stopTime string){
	"stop":  stopAction,
	"start": startAction,
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug(string(body))

	sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
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
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// @bot [action] [agent-name] [duration?]

			params := strings.Split(ev.Text, " ")

			paramsLen := len(params)
			if paramsLen < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else if action, exists := actions[params[1]]; exists {
				action(ev, w, repository.AgentRepository{
					BaseRepository: repository.BaseRepository{
						DB:       *db,
						CommitId: CommitID,
					},
				}, params[2], params[paramsLen-1])
			}
		}
	}
}

func stopAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentName, stopTime string) {
	var until time.Time

	if agent, err := agentRepository.FindAgentByAgentName(agentName); err == nil && agent != nil {
		if until, err = time.Parse("2006-01-02 15:04:05", stopTime); err == nil {
		} else if duration, err := time.ParseDuration(stopTime); err == nil {
			until = time.Now().Add(duration)
		} else {
			until = time.Now().Add(30 * time.Minute)
		}

		agentMarkRepository := repository.AgentMarkRepository{
			BaseRepository: repository.BaseRepository{
				DB:       *db,
				CommitId: CommitID,
			},
		}

		now := time.Now()
		err = agentMarkRepository.Save(repository.AgentMark{
			AgentName: agent.AgentName,
			MarkStart: &now,
		})

		if err != nil {
			msg := fmt.Sprintf("Error occurred while disabling alert %v", err)
			log.Error(errors.New(msg))
			api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		msg := fmt.Sprintf("[알람 중지] \n\n재시작 시점: %s UTC+9 (%v 뒤) \n ", until.Format("2006-01-02 15:04:05"), until.Sub(now).Round(1*time.Minute))
		log.Info(msg)
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		if err != nil {
			log.Error(err)
		}
		return
	} else {
		msg := fmt.Sprintf("Didn't find any agent with received name %s", agentName)
		log.Warn(msg)
		api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false), slack.MsgOptionTS(ev.TimeStamp))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func startAction(ev *slackevents.AppMentionEvent, w http.ResponseWriter, agentRepository repository.AgentRepository, agentName, _ string) {
	var until time.Time

	if agent, err := agentRepository.FindAgentByAgentName(agentName); err != nil && agent != nil {
		agentMarkRepository := repository.AgentMarkRepository{
			BaseRepository: repository.BaseRepository{
				DB:       *db,
				CommitId: CommitID,
			},
		}
		agentMark, err := agentMarkRepository.FindAgentMarkByAgentNameAndTime(agentName, time.Now())
		if err != nil {
			msg := fmt.Sprintf("Error occurred while finding agent mark %v", err)
			log.Error(errors.New(msg))
			api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		now := time.Now()
		agentMark.MarkEnd = &now
		err = agentMarkRepository.Save(*agentMark)

		if err != nil {
			msg := fmt.Sprintf("Error occurred while finding agent mark %v", err)
			log.Error(errors.New(msg))
			api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		msg := fmt.Sprintf("Alert will be disabled until %v", until)
		log.Info(msg)
		api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
		return
	} else {
		msg := botFormatf("Didn't find any agent with received name %s", agentName)

		log.Warn(msg)
		api.PostMessage(ev.Channel, slack.MsgOptionText(msg, false))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func botFormatf(msg string, args ...any) string {
	return fmt.Sprintf("[slack-bot] "+msg, args...)
}
