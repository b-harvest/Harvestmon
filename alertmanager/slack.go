package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/b-harvest/Harvestmon/repository"
	log "github.com/sirupsen/logrus"
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

const InstanceFilter = "instance"

const (
	slkNoteActionId   = "Note"
	slkAckActionId    = "Ack"
	slkStartActionId  = "Start"
	slkStopActionId   = "Stop"
	slkSelectActionId = "Select"
	slkCancelActionId = "Cancel"
	slkLabelsActionId = "Labels"

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

func predefinedHoursSelectBlockElement() slack.BlockElement {

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
	err := alertManager.getConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
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
		alertManager.logger.Fatalf("Error getting slack body: %v", err)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		alertManager.logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if api == nil {
		for _, s := range alertManager.AlarmerConfig.Slacks {
			if eventsAPIEvent.Token == s.VerificationToken {
				api = slack.New(s.BotToken)
				slkCfg = s
			}
		}
		if api == nil {
			w.WriteHeader(http.StatusUnauthorized)
			alertManager.logger.Debug(err.Error())
			return
		}
	}

	sv, err := slack.NewSecretsVerifier(r.Header, slkCfg.SigningSecret)
	if err != nil {
		alertManager.logger.Debug(err.Error())
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
			alertManager.logger.Debug(err.Error())
			return
		} else {
			err = nil
		}
	}

	// prepare repository
	if err != nil {
		alertManager.logger.Error(err.Error())
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
		alertManager.logger.Debug(fmt.Sprintf("URLVerification - challenge: %s", string(cBytes)))
		_, err = w.Write(cBytes)
		if err != nil {
			alertManager.logger.Error(err.Error())
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
					err = nil
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

			if len(params) >= 3 ||
				(len(params) >= 2 && dur == time.Duration(0)) {
				filter := make(map[string]string)
				rawFilterStartIdx := 2
				if len(params) >= 2 && dur == time.Duration(0) {
					rawFilterStartIdx = 1

				}
				for i := rawFilterStartIdx; i < len(params); i++ {
					rawFilter := strings.Split(params[i], "=")
					if len(rawFilter) == 2 {
						filter[rawFilter[0]] = rawFilter[1]
					}
				}
				if dur, err = time.ParseDuration(params[len(params)-1]); err != nil {
					dur = time.Minute * 30
					err = nil
				}

				var msg string

				now := time.Now()
				filteredAgents, err := alertManager.reader.FindAgentByLabel(filter)
				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				var instanceNames []string
				for _, agent := range filteredAgents {
					instanceNames = append(instanceNames, agent.Instance)
				}
				if action == slkStopActionId {
					endTime := now.Add(dur)
					userId := ev.User

					var marks []repository.StoreEntity
					for _, instance := range instanceNames {
						marks = append(marks, &repository.AgentMark{
							Instance:           instance,
							MarkStart:          &now,
							MarkEnd:            &endTime,
							MarkerUserIdentity: userId,
							MarkerFrom:         MARKER_FROM,
						})
					}
					err = alertManager.writer.SaveAll(marks)
					if err != nil {
						_, _, err = api.PostMessage(chanId, slack.MsgOptionText(fmt.Sprintf("%s failed to save agentMark: %s", failedEmoticon, err.Error()), false), slack.MsgOptionTS(ev.TimeStamp))
						return
					}

					msg = formatStopAlarmFormat(dur, ev.User)
				} else {

					var shouldDeleteMarks []repository.StoreEntity
					for _, instance := range instanceNames {
						agentMarks, err := alertManager.reader.FindAgentMarkByInstanceAndTime(instance, &now)
						if err != nil {
							alertManager.logger.Error(err.Error())
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						for _, mark := range agentMarks {
							mark.MarkEnd = &now
							shouldDeleteMarks = append(shouldDeleteMarks, mark)

						}
					}

					err = alertManager.writer.SaveAll(shouldDeleteMarks)
					if err != nil {
						alertManager.logger.Error(err.Error())
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					instanceNames = nil
					for _, mark := range shouldDeleteMarks {
						instanceNames = append(instanceNames, mark.(*repository.AgentMark).Instance)
					}

					msg = formatStartAlarmFormat(ev.User)
				}

				filterMsg := ""
				for k, v := range filter {
					if filterMsg != "" {
						filterMsg = fmt.Sprintf("%s\n", filterMsg)
					}
					filterMsg = fmt.Sprintf("%s- %s=%s", filterMsg, k, v)
				}

				msgOptions = append(msgOptions, slack.MsgOptionBlocks(slack.NewSectionBlock(&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("%s\n"+
						"Filter\n"+
						" %s\n"+
						"Agents\n "+
						"- %s\n", msg, filterMsg, strings.Join(instanceNames, "\n- ")),
				}, nil, nil)), slack.MsgOptionTS(ev.TimeStamp))
			} else {

				var (
					msg     string
					options []string
				)
				switch {
				case action == slkStartActionId:
					msg = fmt.Sprintf("select agent to start alarm")
					agents, err := alertManager.reader.FindAgents()
					if err != nil {
						alertManager.logger.Error(err.Error())
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					now := time.Now()
					for _, agent := range agents {
						agentMarks, err := alertManager.reader.FindAgentMarkByInstanceAndTime(agent.Instance, &now)
						if err != nil {
							alertManager.logger.Warningf(err.Error())
							continue
						}
						if len(agentMarks) > 0 {
							options = append(options, agent.Instance)
						}
					}

				case action == slkStopActionId:
					msg = fmt.Sprintf("select agent to stop alarm")
					agents, err := alertManager.reader.FindAgents()
					if err != nil {
						alertManager.logger.Error(err.Error())
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					for _, agent := range agents {
						options = append(options, agent.Instance)
					}
				}

				if len(options) > 0 {
					msgOptions, err = selectAction(ev, msg, options, action, dur)
				} else {
					msgOptions = []slack.MsgOption{slack.MsgOptionText("no available options", false), slack.MsgOptionTS(ev.TimeStamp)}
				}

				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			_, _, err = api.PostMessage(chanId, msgOptions...)
			if err != nil {
				alertManager.logger.Error(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		}
		return
	case string(slack.InteractionTypeInteractionMessage): //  user interacted with `interactionMessage`

		var interactionCallback slack.InteractionCallback
		err = json.Unmarshal(body, &interactionCallback)
		if err != nil {
			alertManager.logger.Error(err.Error())
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
				agent, err := alertManager.reader.FindAgentByInstance(agentName)
				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
				} else if agent == nil {
					alertManager.logger.Warningf("agent %s not found", agentName)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				now := time.Now()
				agentMarks, err := alertManager.reader.FindAgentMarkByInstanceAndTime(agentName, &now)

				var shouldStopAgentMarks []repository.StoreEntity
				for _, mark := range agentMarks {
					if mark.Instance == agent.Instance && mark.MarkStart != nil && mark.MarkerUserIdentity != "" {
						mark.MarkEnd = &now
						shouldStopAgentMarks = append(shouldStopAgentMarks, mark)
					}
				}
				err = alertManager.writer.SaveAll(shouldStopAgentMarks)
				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				msg = fmt.Sprintf("%s\n", agentName)

				actionName := callbackId[strings.Index(callbackId, ",")+1 : strings.LastIndex(callbackId, ",")]
				duration, err := time.ParseDuration(callbackId[strings.LastIndex(callbackId, ",")+1:])
				if err != nil || duration == time.Duration(0) {
					duration = 30 * time.Minute
				}

				switch actionName {
				case slkStartActionId:

					alrms, err := alertManager.reader.FindActiveAlarmsByInstance(agentName)
					err = alertManager.writer.DeleteActiveAlarms(alrms)
					if err != nil {
						alertManager.logger.Error(err.Error())
					}

					msg += formatStartAlarmFormat(userId)
					break
				case slkStopActionId:

					endTime := now.Add(duration)
					err = alertManager.writer.Save(
						&repository.AgentMark{
							Instance:           agentName,
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
			alertManager.logger.Error(err.Error())
		}
		return
	case string(slack.InteractionTypeBlockActions): // user interacted with `blockAction`
		var interactionCallback slack.InteractionCallback
		err = json.Unmarshal(body, &interactionCallback)
		if err != nil {
			alertManager.logger.Error(err.Error())
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
				ae               = alertEventFromMessageBlocks(blockSet)
			)

			if firstBlockAction.Text.Text == slkStartActionId {
				//replaceButton(&interactionCallback, slkStartActionId, slkAckActionId)
				replacedEmoticon = slkLargeGreenCircleEmoticon
				msg = formatStartAlarmFormat(userId)
				blockSet, _ = removeButtons(blockSet)
				alrts, err := alertManager.writer.FindAlertRecordsByInstanceAndResolvTimestamp(agentName, nil)
				for _, alrt := range alrts {
					if alrt.AlertEvent == ae {
						err = alertManager.writer.UpdateResolvTs(alrt, now)
						break
					}
				}
				if err != nil {
					alertManager.logger.Error(err.Error())
				}

				alrms, err := alertManager.reader.FindActiveAlarmsByInstance(agentName)
				var toDeleteAlrms []repository.ActiveAlarm
				for _, alrm := range alrms {
					if alarmName(alrm.AlarmerName).GetAlertEvent() == alertEvent(ae) {
						toDeleteAlrms = append(toDeleteAlrms, alrm)
					}
				}
				err = alertManager.writer.DeleteActiveAlarms(toDeleteAlrms)
				if err != nil {
					alertManager.logger.Error(err.Error())
				}

				if ae != "" {
					resolvePagerDuty(agentName, ae)
				}

				agentMarks, err := alertManager.reader.FindAgentMarkByInstanceAndTime(agentName, &now)

				var shouldDeleteMarks []repository.StoreEntity
				for _, mark := range agentMarks {
					if mark.Instance == agentName && mark.MarkStart != nil && mark.MarkerUserIdentity != "" {
						mark.MarkEnd = &now
						shouldDeleteMarks = append(shouldDeleteMarks, mark)
					}
				}

				err = alertManager.writer.SaveAll(shouldDeleteMarks)
				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				blockSet = removeContainingMessages(blockSet, "disabled alert until to")
				blockSet = removeContainingMessages(blockSet, "Filter")

			} else {
				replacedEmoticon = slkConstructionEmoticon

				interactionValue := firstBlockAction.SelectedOption.Value // e.g. 30m

				if markDuration, err = time.ParseDuration(interactionValue); err != nil {
					markDuration = time.Minute * 30 // default value
				}

				// replace button from `ack` to `start`
				blockSet, _ = replaceButton(blockSet, slkAckActionId, slkStartActionId)
				until := now.Add(markDuration)
				agentMark := &repository.AgentMark{
					MarkStart:          &now,
					MarkEnd:            &until,
					MarkerUserIdentity: userId,
					MarkerFrom:         MARKER_FROM,
					Instance:           agentName,
				}

				// remove agentMark
				err = endAgentMarks(alertManager.writer, agentName)

				err = alertManager.writer.Save(agentMark)
				if err != nil {
					alertManager.logger.Error(err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				msg = formatStopAlarmFormat(markDuration, userId)
				blockSet = removeContainingMessages(blockSet, "disabled alert until to")
				blockSet = removeContainingMessages(blockSet, "Filter")

				if ae != "" {
					acknowledgePagerDuty(agentName, ae)
				}
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
			alertManager.logger.Error(err.Error())
		}

		return
	}

}

// alertEventFromMessageBlocks extracts the AlertEvent name embedded in the alert
// message body (blockSet[1], built by alertMessageBlocks with the format
// "service: %s\nalert: %s\n\n%s") so Slack button handlers can target the right
// PagerDuty incident. Returns "" if it can't find a match.
func alertEventFromMessageBlocks(blockSet []slack.Block) string {
	if len(blockSet) < 2 || blockSet[1].BlockType() != slack.MBTSection {
		return ""
	}
	section, ok := blockSet[1].(*slack.SectionBlock)
	if !ok || section.Text == nil {
		return ""
	}

	const serviceKey = "service: "
	body := section.Text.Text
	idx := strings.Index(body, serviceKey)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(serviceKey):]
	if end := strings.Index(rest, "\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// acknowledgePagerDuty tells every enabled PagerDuty alarmer that the incident
// for instance+alertEvent has been acknowledged from Slack.
func acknowledgePagerDuty(instance, ae string) {
	for _, pdConf := range alertManager.AlarmerConfig.Pagerdutys {
		if !pdConf.Enabled {
			continue
		}
		if err := pdConf.acknowledge(InstanceName(instance), alertEvent(ae)); err != nil {
			alertManager.logger.Errorf("failed to acknowledge PagerDuty incident for %s/%s: %v", instance, ae, err)
		}
	}
}

// resolvePagerDuty tells every enabled PagerDuty alarmer that the incident for
// instance+alertEvent has been manually resolved from Slack.
func resolvePagerDuty(instance, ae string) {
	for _, pdConf := range alertManager.AlarmerConfig.Pagerdutys {
		if !pdConf.Enabled {
			continue
		}
		if err := pdConf.resolve(InstanceName(instance), alertEvent(ae)); err != nil {
			alertManager.logger.Errorf("failed to resolve PagerDuty incident for %s/%s: %v", instance, ae, err)
		}
	}
}

func extractURLWithPrefix(input string) string {
	strt := strings.Index(input, "<")
	end := strings.Index(input, ">")

	if strt != -1 && end != -1 && strt < end {
		pipe := strings.Index(input[strt:end], "|")
		if pipe != -1 {
			prefix := input[:strt]
			origin := input[strt+pipe+1 : end]

			return prefix + origin
		}
	}

	// Return the original string if <> or | not found
	return input
}

func selectAction(ev *slackevents.AppMentionEvent, msg string, options []string, secondActionId string, markDuration time.Duration) ([]slack.MsgOption, error) {

	var attachmentActionOptions []slack.AttachmentActionOption

	for _, option := range options {
		attachmentActionOptions = append(attachmentActionOptions, slack.AttachmentActionOption{
			Text:  option,
			Value: option,
		})
	}

	attachment := slack.Attachment{
		Text:       msg,
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
	agent, err := repo.FindAgentByInstance(agentName)
	agentMarks, err := repo.FindAgentMarkByInstanceAndTime(agentName, &now)
	if err != nil {
		return errors.New("failed to find marks for agent " + agentName)
	}

	if agent == nil {
		return errors.New("agent " + agentName + " not found")
	}

	if len(agentMarks) == 0 {
		return nil
	}

	var shouldDeleteMarks []repository.StoreEntity
	for _, mark := range agentMarks {
		if mark.Instance == agentName {
			mark.MarkEnd = &now
			shouldDeleteMarks = append(shouldDeleteMarks, mark)
			break
		}
	}
	err = repo.SaveAll(shouldDeleteMarks)
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

func removeContainingMessages(blockset []slack.Block, substr string) []slack.Block {
	var summarizedBlocks []slack.Block
	for _, block := range blockset {
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
	return summarizedBlocks
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
