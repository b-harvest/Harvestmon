package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const MARKER_FROM = "slack"

const (
	slkNoteActionId   = "Note"
	slkAckActionId    = "Ack"
	slkStartActionId  = "Start"
	slkStopActionId   = "Stop"
	slkSelectActionId = "Select"
	slkCancelActionId = "Cancel"

	slkStopWatchEmoticon        = ":stopwatch:"
	slkConstructionEmoticon     = ":construction:"
	slkLargeGreenCircleEmoticon = ":large_green_circle:"
	slkWhiteCheckMarkEmoticon   = ":white_check_mark:"

	failedEmoticon = ":x:"
)

var slkPredefineHours = []time.Duration{
	30 * time.Minute,
	time.Hour,
	2 * time.Hour,
	3 * time.Hour,
	6 * time.Hour,
	12 * time.Hour,
	24 * time.Hour,
	2 * 24 * time.Hour,
	7 * 24 * time.Hour,
}

func newPredefinedHoursSelectBlockElement() slack.BlockElement {

	var attachmentActionOptions []*slack.OptionBlockObject
	for _, ph := range slkPredefineHours {
		attachmentActionOptions = append(attachmentActionOptions, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: ph.String(),
			},
			Value: ph.String(),
		})
	}
	return &slack.SelectBlockElement{
		Type:    "static_select",
		Options: attachmentActionOptions,
	}

}

func restHandler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	handleSlack(rr, req)

	if rr.StatusCode == 0 {
		rr.StatusCode = http.StatusOK
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

func handleSlack(w http.ResponseWriter, r *http.Request) {
	cc, err := configManager.GetConfig()
	if err != nil {
		configManager.logger.Fatalf("Error loading configuration: %v", err)
		return
	}

	var (
		api    *slack.Client
		body   []byte
		slkCfg SlackConfig
	)

	// parse request body
	body, err = getSlkBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		configManager.logger.Fatalf("Error getting slack body: %v", err)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if api == nil {
		for _, s := range cc.AlarmerConfig.Slacks {
			if eventsAPIEvent.Token == s.VerificationToken {
				api = slack.New(s.BotToken)
				slkCfg = s
			}
		}
		if api == nil {
			w.WriteHeader(http.StatusUnauthorized)
			cc.logger.Debug(err.Error())
			return
		}
	}

	sv, err := slack.NewSecretsVerifier(r.Header, slkCfg.SigningSecret)
	if err != nil {
		cc.logger.Debug(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err = sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = sv.Ensure(); err != nil {
		if eventsAPIEvent.Token != slkCfg.VerificationToken {
			w.WriteHeader(http.StatusUnauthorized)
			cc.logger.Debug(err.Error())
			return
		}
	}

	// prepare repository
	if err != nil {
		cc.logger.Debug(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		var challengeRes *slackevents.ChallengeResponse
		err = json.Unmarshal([]byte(body), &challengeRes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		challenge := map[string]string{
			"challenge": challengeRes.Challenge,
		}
		cBytes, _ := json.Marshal(challenge)
		cc.logger.Debug(fmt.Sprintf("URLVerification - challenge: %s", string(cBytes)))
		_, err = w.Write(cBytes)
		if err != nil {
			cc.logger.Error(err.Error())
		}
		break
	case slackevents.CallbackEvent: // user mention bot
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// @bot [action] [duration?]
			metion := regexp.MustCompile(`<@[A-Z0-9]+>`).FindString(ev.Text)

			params := strings.Split(
				strings.TrimPrefix(ev.Text, ev.Text[:strings.Index(ev.Text, metion)+len(metion)+1]),
				" ")

			if len(params) < 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var (
				msgOptions []slack.MsgOption
				action     string
				dur        time.Duration
				chanId     = ev.Channel
			)

			if len(params) >= 2 {
				dur, err = time.ParseDuration(params[1])
				if err != nil {
					dur = time.Duration(0)
				}
			} else {
				dur = time.Duration(0)
			}

			switch strings.ToLower(params[0]) {
			case strings.ToLower(slkStartActionId):
				action = slkStartActionId
			case strings.ToLower(slkStopActionId):
				action = slkStopActionId
			default:

				w.WriteHeader(http.StatusBadRequest)
				return
			}

			msgOptions, err = selectAction(ev, cc.writeRepo, action, dur)
			if err != nil {
				cc.logger.Error(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_, _, err = api.PostMessage(chanId, msgOptions...)
			if err != nil {
				cc.logger.Error(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		}
		return
	case string(slack.InteractionTypeInteractionMessage): //  user interacted with `interactionMessage`

		var interactionCallback slack.InteractionCallback
		err = json.Unmarshal(body, &interactionCallback)
		if err != nil {
			cc.logger.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(interactionCallback.ActionCallback.AttachmentActions) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		callBackAction := interactionCallback.ActionCallback.AttachmentActions[0]

		var (
			msg        string
			agentName  string
			chanId     = interactionCallback.Channel.ID
			userId     = interactionCallback.User.ID
			callbackId = interactionCallback.CallbackID
			ts         = interactionCallback.Message.Msg.Timestamp
		)

		if ts == "" {
			ts = interactionCallback.Message.Msg.ThreadTimestamp
			if ts == "" {
				ts = interactionCallback.MessageTs
			}
		}

		switch strings.ToLower(callBackAction.Name) {
		case strings.ToLower(slkSelectActionId):
			{
				agentName = extractURLWithPrefix(interactionCallback.ActionCallback.AttachmentActions[0].SelectedOptions[0].Value)
				now := time.Now()
				agentMarks, err := cc.readRepo.FindAgentMarkByAgentNameAndTime(agentName, now)
				if err != nil || len(agentMarks) == 0 {
				} else {
					for _, am := range agentMarks {
						if am.AgentName != "" && am.MarkStart != nil && am.MarkerUserIdentity != "" {
							am.MarkEnd = &now
							err = cc.writeRepo.Save(am)
						}
					}
				}

				msg = fmt.Sprintf("%s\n", agentName)

				actionName := callbackId[strings.Index(callbackId, ",")+1 : strings.LastIndex(callbackId, ",")]
				duration, err := time.ParseDuration(callbackId[strings.LastIndex(callbackId, ",")+1:])
				if err != nil || duration == time.Duration(0) {
					duration = 30 * time.Minute
				}

				switch actionName {
				case slkStartActionId:
					msg += formatStartAlarmFormat(userId)
					break
				case slkStopActionId:

					endTime := now.Add(duration)
					err = cc.writeRepo.Save(
						repository.AgentMark{
							AgentName:          agentName,
							MarkStart:          &now,
							MarkEnd:            &endTime,
							MarkerUserIdentity: userId,
							MarkerFrom:         MARKER_FROM,
						})
					if err != nil {
						_, _, err = api.PostMessage(chanId, slack.MsgOptionText(fmt.Sprintf("%s failed to save agentMark: %s", failedEmoticon, err.Error()), false), slack.MsgOptionTS(ts))
						return
					}
					msg += formatStopAlarmFormat(duration, userId)
					break
				}

			}
		case strings.ToLower(slkCancelActionId):
			{
				_, _, err = api.DeleteMessage(chanId, ts)
				return
			}
		}

		if msg == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		interactionCallback.Message.Msg.Blocks.BlockSet = append(interactionCallback.Message.Msg.Blocks.BlockSet,
			slack.NewContextBlock("", slack.TextBlockObject{
				Type:  slack.MarkdownType,
				Text:  msg,
				Emoji: false,
			}))

		_, _, err = api.PostMessage(chanId, slack.MsgOptionBlocks(interactionCallback.Message.Msg.Blocks.BlockSet...), slack.MsgOptionTS(ts))
		_, _, err = api.DeleteMessage(chanId, ts)
		if err != nil {
			cc.logger.Error(err.Error())
		}
		return
	case string(slack.InteractionTypeBlockActions): // user interacted with `blockAction`
		var interactionCallback slack.InteractionCallback
		err = json.Unmarshal(body, &interactionCallback)
		if err != nil {
			cc.logger.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var (
			markDuration time.Duration
			blockActions = interactionCallback.ActionCallback.BlockActions
			blockSet     = interactionCallback.Message.Msg.Blocks.BlockSet
		)

		if len(blockActions) < 1 ||
			len(blockSet) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		firstBlock := blockSet[0]
		if firstBlock.BlockType() != slack.MBTSection {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		firstBlockAction := blockActions[0]

		var (
			ts              = interactionCallback.Message.Msg.Timestamp
			userId          = interactionCallback.User.ID
			agentName, _, _ = strings.Cut(extractURLWithPrefix(firstBlock.(*slack.SectionBlock).Text.Text[strings.LastIndex(firstBlock.(*slack.SectionBlock).Text.Text, " ")+1:]), "*")
			chanId          = interactionCallback.Channel.ID
		)

		now := time.Now()

		// if user responses with `slkNoteActionId` button, and there is no user input,
		if firstBlockAction.Text.Text == slkNoteActionId {
			blockSet = append(blockSet,
				slack.NewInputBlock("",
					slack.NewTextBlockObject(slack.PlainTextType, fmt.Sprintf("<@%s> please note to here.", userId), false, false),
					nil,
					slack.NewPlainTextInputBlockElement(nil, slkNoteActionId),
				).WithDispatchAction(true))

			// if user responses with `slkNoteActionId` button, but also there is user input,
		} else if firstBlockAction.ActionID == slkNoteActionId {

			if firstBlockAction.Value != "" {

				var notInputBlocks []slack.Block
				for _, block := range blockSet {
					if block.BlockType() != slack.MBTInput {
						notInputBlocks = append(notInputBlocks, block)
					}
				}

				blockSet = append(notInputBlocks,
					slack.NewContextBlock("", slack.TextBlockObject{
						Type:  slack.MarkdownType,
						Text:  fmt.Sprintf("<@%s>: %s", userId, firstBlockAction.Value),
						Emoji: false,
					}),
				)
			}
		} else { // if user doesn't response with `slkNoteActionId`, make it as disabledAlert action.

			var (
				msg              string
				replacedEmoticon string
				ae               string
			)

			// remove agentMark
			err = endAgentMarks(cc.writeRepo, agentName)

			if firstBlockAction.Text.Text == slkStartActionId {
				//replaceButton(&interactionCallback, slkStartActionId, slkAckActionId)
				replacedEmoticon = slkLargeGreenCircleEmoticon
				msg = formatStartAlarmFormat(userId)
				blockSet, _ = removeButtons(blockSet)
				alrts, err := cc.writeRepo.FindAlertRecordsByNodeNameAndResolvTimestamp(agentName, nil)
				for _, alrt := range alrts {
					if len(blockSet) < 2 ||
						blockSet[1].BlockType() != slack.MBTSection ||
						len(blockSet[1].(*slack.SectionBlock).Fields) < 1 {
						continue
					}
					alertBody := blockSet[1].(*slack.SectionBlock).Fields[0].Text
					aeKey := "service: "
					ae = alertBody[strings.Index(alertBody, aeKey)+len(aeKey) : strings.Index(alertBody, "\n")]
					if alrt.AlertEvent == ae {
						err = cc.writeRepo.UpdateResolvTs(alrt, now)
						break
					}
				}
				if err != nil {
					cc.logger.Error(err.Error())
				}

				alrms, err := cc.readRepo.FindActiveAlarmsByNodeName(agentName)
				var toDeleteAlrms []repository.ActiveAlarm
				for _, alrm := range alrms {
					if alarmName(alrm.AlarmerName).GetAlertEvent() == alertEvent(ae) {
						toDeleteAlrms = append(toDeleteAlrms, alrm)
					}
				}
				err = cc.writeRepo.DeleteActiveAlarms(toDeleteAlrms)
				if err != nil {
					cc.logger.Error(err.Error())
				}

				agentMarks, err := cc.readRepo.FindAgentMarkByAgentNameAndTime(agentName, now)
				if err != nil || len(agentMarks) == 0 {
				} else {
					for _, am := range agentMarks {
						if am.AgentName != "" && am.MarkStart != nil && am.MarkerUserIdentity != "" {
							am.MarkEnd = &now
							err = cc.writeRepo.Save(am)
						}
					}
				}

			} else {
				replacedEmoticon = slkConstructionEmoticon
				if firstBlockAction.Text.Text == slkCancelActionId {
					markDuration = time.Minute * 30
				} else {
					interactionValue := firstBlockAction.SelectedOption.Value // e.g. 30m

					if markDuration, err = time.ParseDuration(interactionValue); err != nil {
						markDuration = time.Minute * 30 // default value
					}

				}
				// replace button from `ack` to `start`
				blockSet, _ = replaceButton(blockSet, slkAckActionId, slkStartActionId)
				until := now.Add(markDuration)
				agentMark := repository.AgentMark{
					AgentName:          agentName,
					MarkStart:          &now,
					MarkEnd:            &until,
					MarkerUserIdentity: userId,
					MarkerFrom:         MARKER_FROM,
				}

				err = cc.writeRepo.Save(agentMark)
				if err != nil {
					cc.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				msg = formatStopAlarmFormat(markDuration, userId)
				removeContainingMessages(&interactionCallback, "disabled alert until to")
			}
			firstBlock.(*slack.SectionBlock).Text.Text = replaceColonToString(firstBlock.(*slack.SectionBlock).Text.Text, replacedEmoticon)

			blockSet = append(blockSet,
				slack.NewContextBlock("", slack.TextBlockObject{
					Type:  slack.MarkdownType,
					Text:  msg,
					Emoji: false,
				}))

		}

		_, _, _, err = api.UpdateMessage(chanId, ts, slack.MsgOptionBlocks(blockSet...))
		if err != nil {
			cc.logger.Error(err.Error())
		}

		return
	}

}

func extractURLWithPrefix(input string) string {
	start := strings.Index(input, "<")
	end := strings.Index(input, ">")

	if start != -1 && end != -1 && start < end {
		pipe := strings.Index(input[start:end], "|")
		if pipe != -1 {
			prefix := input[:start]
			origin := input[start+pipe+1 : end]

			return prefix + origin
		}
	}

	// Return the original string if <> or | not found
	return input
}

func selectAction(ev *slackevents.AppMentionEvent, repo *repository.Repository, secondActionId string, markDuration time.Duration) ([]slack.MsgOption, error) {
	var (
		err error
	)

	agents, err := repo.FindAgentsAll()
	if err != nil {
		return nil, err
	}

	var attachmentActionOptions []slack.AttachmentActionOption

	for _, agent := range agents {
		attachmentActionOptions = append(attachmentActionOptions, slack.AttachmentActionOption{
			Text:  agent.AgentName,
			Value: agent.AgentName,
		})
	}

	attachment := slack.Attachment{
		Text:       fmt.Sprintf("choose agent to stop alarm"),
		CallbackID: fmt.Sprintf("%s,%s,%s", slkSelectActionId, secondActionId, markDuration),
		Actions: []slack.AttachmentAction{
			{
				Name:    slkSelectActionId,
				Text:    slkSelectActionId,
				Type:    "select",
				Options: attachmentActionOptions,
			},
			{
				Name:  slkCancelActionId,
				Text:  slkCancelActionId,
				Type:  "button",
				Style: "danger",
			},
		},
	}

	return []slack.MsgOption{slack.MsgOptionTS(ev.TimeStamp), slack.MsgOptionAttachments(attachment)}, nil
}

func endAgentMarks(repo *repository.Repository, agentName string) error {
	now := time.Now()
	agentMarks, err := repo.FindAgentMarkByAgentNameAndTime(agentName, now)
	if err != nil || len(agentMarks) == 0 {
		return errors.New("no agent marks found")
	}

	for _, agentMark := range agentMarks {
		agentMark.MarkEnd = &now
		err = repo.Save(agentMark)
	}
	if err != nil {
		return err
	}
	return nil
}

func getSlkBody(r *http.Request) ([]byte, error) {
	var (
		body []byte
		err  error
	)
	if r.Header.Get("Content-Type") != "application/json" {
		// Parse form data
		if err = r.ParseForm(); err != nil {
			return nil, err
		}

		// Extract the payload
		payload := r.PostFormValue("payload")

		// Decode the URL-encoded payload
		decodedPayload, err := url.QueryUnescape(payload)
		if err != nil {
			return nil, err
		}
		r.Header.Set("Content-Type", "application/json")
		body = []byte(decodedPayload)
	} else {
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

func replaceColonToString(input, replace string) string {
	// Regular expression to match words wrapped with ":"
	re := regexp.MustCompile(`:[a-zA-Z0-9_]+:`)

	// Replace all matches with ":construction:"
	result := re.ReplaceAllString(input, replace)
	return result
}

func replaceButton(blockSet []slack.Block, from, to string) ([]slack.Block, int) {
	var cnt int
	for _, b := range blockSet {
		if b.BlockType() == slack.MBTAction {
			for _, e := range b.(*slack.ActionBlock).Elements.ElementSet {
				if e.ElementType() == slack.METButton {
					button := e.(*slack.ButtonBlockElement)
					if button.Text.Text == from {
						button.Value = to
						button.Text.Text = to
						cnt++
					}
				}

			}
		}
	}
	return blockSet, cnt
}

func removeButtons(blockSet []slack.Block) ([]slack.Block, int) {
	var (
		cnt             int
		notActionBlocks []slack.Block
	)
	for _, b := range blockSet {
		if b.BlockType() != slack.MBTAction {
			notActionBlocks = append(notActionBlocks, b)
			cnt++
		}
	}
	blockSet = notActionBlocks
	return blockSet, cnt
}

func removeContainingMessages(interactionCallback *slack.InteractionCallback, substr string) {
	var summarizedBlocks []slack.Block
	for _, block := range interactionCallback.Message.Msg.Blocks.BlockSet {
		removeIt := false
		if block.BlockType() == slack.MBTContext {
			for _, e := range block.(*slack.ContextBlock).ContextElements.Elements {
				if e.MixedElementType() == slack.MixedElementText {
					if strings.Contains(e.(*slack.TextBlockObject).Text, substr) {
						removeIt = true
						break
					}
				}
			}
		}
		if !removeIt {
			summarizedBlocks = append(summarizedBlocks, block)
		}
	}
	interactionCallback.Message.Msg.Blocks.BlockSet = summarizedBlocks
}

func formatStartAlarmFormat(userId string) string {
	return fmt.Sprintf("%s start alert from *%s* by <@%s>",
		slkWhiteCheckMarkEmoticon,
		time.Now().Format(time.DateTime),
		userId)

}

func formatStopAlarmFormat(markDuration time.Duration, userId string) string {
	return fmt.Sprintf("%s disabled alert until to *%s UTC* by <@%s> (%v)",
		slkStopWatchEmoticon,
		time.Now().Add(markDuration).Format(time.DateTime),
		userId,
		markDuration.String())
}
