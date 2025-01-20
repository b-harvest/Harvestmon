package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/b-harvest/Harvestmon/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/slack-go/slack"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AlertLevel is alias type of string to deal with level to prevent confusion.
type AlertLevel string

func (a *AlertLevel) String() string {
	return string(*a)
}

func (a *AlertLevel) checkValid() error {
	if a.String() == "" {
		return errors.New("alert level cannot be empty")
	}
	return nil
}

// AlarmerConfig is struct for multiple platform alarm configurations
type AlarmerConfig struct {
	// Pagerdutys configurations
	Pagerdutys []PDConfig `toml:"pagerdutys"`
	// Discords webhook configurations
	Discords []DiscordConfig `toml:"discords"`
	// Telegrams webhook configurations
	Telegrams []TelegramConfig `toml:"telegrams"`
	// Slacks webhook configurations
	Slacks []SlackConfig `toml:"slacks"`
}

func (c *AlarmerConfig) getAlarmers() []Alarmer {
	alarmers := []Alarmer{}
	for _, a := range c.Pagerdutys {
		alarmers = append(alarmers, &a)
	}
	for _, a := range c.Discords {
		alarmers = append(alarmers, &a)
	}
	for _, a := range c.Telegrams {
		alarmers = append(alarmers, &a)
	}
	for _, a := range c.Slacks {
		alarmers = append(alarmers, &a)
	}

	return alarmers
}

// Alarmer is interface to define what alertLevel Alarmer should be invoked.
type Alarmer interface {
	getName() string
	getTargetAlertLevels() []AlertLevel
	getResendDuration() time.Duration

	shouldSendResolveMsg() bool
	notify(msg *alertMsg) (int64, error) // returns sentTime, error
}

// PDConfig is the information required to send alerts to PagerDuty
type PDConfig struct {
	Enabled           bool          `toml:"enabled"`
	ApiKey            string        `toml:"apiKey"`
	DefaultSeverity   string        `toml:"defaultSeverity"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`
}

func (c *PDConfig) getName() string {
	return string(pd)
}

func (c *PDConfig) getTargetAlertLevels() []AlertLevel {
	return c.TargetAlertLevels
}

func (c *PDConfig) getResendDuration() time.Duration {
	return c.ResendDuration
}

func (c *PDConfig) shouldSendResolveMsg() bool {
	return c.ResolveMsg
}

func (c *PDConfig) notify(msg *alertMsg) (int64, error) {
	action := "trigger"
	if msg.resolved {
		action = "resolve"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: c.ApiKey,
		Action:     action,
		DedupKey:   msg.uniqueId,
		Payload: &pagerduty.V2Payload{
			Summary:  msg.message,
			Source:   msg.uniqueId,
			Severity: c.DefaultSeverity,
		},
	})
	return time.Now().Unix() * 1e6, err
}

// DiscordConfig holds the information needed to publish to a Discord webhook for sending alerts
type DiscordConfig struct {
	Enabled           bool          `toml:"enabled"`
	Webhook           string        `toml:"webhook"`
	Mentions          []string      `toml:"mentions"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`
}

func (c *DiscordConfig) getName() string {
	return string(di)
}

func (c *DiscordConfig) getTargetAlertLevels() []AlertLevel {
	return c.TargetAlertLevels
}

func (c *DiscordConfig) getResendDuration() time.Duration {
	return c.ResendDuration
}

func (c *DiscordConfig) shouldSendResolveMsg() bool {
	return c.ResolveMsg
}

func (c *DiscordConfig) notify(msg *alertMsg) (int64, error) {
	discPost := buildDiscordMessage(msg)
	client := &http.Client{}
	data, err := json.MarshalIndent(discPost, "", "  ")
	if err != nil {
		return 0, errors.New("failed to marshal discord post body" + err.Error())
	}

	req, err := http.NewRequest("POST", c.Webhook, bytes.NewBuffer(data))
	if err != nil {
		return 0, errors.New("failed to create discord post request" + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.New("failed to send discord post request" + err.Error())
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 204 {
		return 0, errors.New(fmt.Sprintf("discord post request failed with status code %d, %s", resp.StatusCode, resp.Body))
	}
	return time.Now().Unix() * 1e6, nil
}

type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	AvatarUrl string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Url         string `json:"url,omitempty"`
	Description string `json:"description"`
	Color       uint   `json:"color"`
}

func buildDiscordMessage(msg *alertMsg) *DiscordMessage {
	prefix := ""
	if msg.resolved {
		prefix = "🟢 Resolved: "
	} else {
		prefix = "🔴 Warning: "
	}

	return &DiscordMessage{
		Username: "Harvestmon",
		Content:  prefix + msg.node,
		Embeds: []DiscordEmbed{{
			Description: msg.message,
		}},
	}
}

// TelegramConfig holds the information needed to publish to a Telegram webhook for sending alerts
type TelegramConfig struct {
	Enabled           bool          `toml:"enabled"`
	ApiKey            string        `toml:"apiKey"`
	Channel           string        `toml:"channel"`
	Mentions          []string      `toml:"mentions"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`
}

func (c *TelegramConfig) getName() string {
	return string(tg)
}

func (c *TelegramConfig) getTargetAlertLevels() []AlertLevel {
	return c.TargetAlertLevels
}

func (c *TelegramConfig) getResendDuration() time.Duration {
	return c.ResendDuration
}

func (c *TelegramConfig) shouldSendResolveMsg() bool {
	return c.ResolveMsg
}

func (c *TelegramConfig) notify(msg *alertMsg) (int64, error) {
	bot, err := tgbotapi.NewBotAPI(c.ApiKey)
	if err != nil {
		return 0, errors.New("notify telegram: " + err.Error())
	}

	prefix := ""
	if msg.resolved {
		prefix = "🟢 Resolved: "
	} else {
		prefix = "🔴 Warning: "
	}

	mc := tgbotapi.NewMessageToChannel(c.Channel, fmt.Sprintf("%s %s\n\n%s", prefix, msg.node, msg.message))
	_, err = bot.Send(mc)
	if err != nil {
		return 0, errors.New("notify telegram: " + err.Error())
	}
	return time.Now().Unix() * 1e6, nil
}

// SlackConfig holds the information needed to publish to a Slack webhook for sending alerts
type SlackConfig struct {
	Enabled           bool          `toml:"enabled"`
	BotToken          string        `toml:"botToken"`
	Channel           string        `toml:"channel"`
	Mentions          []string      `toml:"mentions"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`

	Interactive       bool   `toml:"interactive"`
	VerificationToken string `toml:"verificationToken"`
	SigningSecret     string `toml:"signingSecret"`

	api *slack.Client
}

func (c *SlackConfig) getName() string {
	return string(slk)
}

func (c *SlackConfig) getTargetAlertLevels() []AlertLevel {
	return c.TargetAlertLevels
}

func (c *SlackConfig) getResendDuration() time.Duration {
	return c.ResendDuration
}

func (c *SlackConfig) shouldSendResolveMsg() bool {
	return c.ResolveMsg
}

func (c *SlackConfig) notify(msg *alertMsg) (int64, error) {
	if c.api == nil {
		c.api = slack.New(c.BotToken)
	}

	var (
		err           error
		content       []slack.MsgOption
		unixTs        float64
		respTimestamp string
	)
	if c.Interactive {
		content = buildInteractiveSlackMessage(msg, strings.Join(c.Mentions, ", "))
		if msg.resolved && msg.lastSentTime != 0 {
			ts := fmt.Sprintf("%f", float64(msg.lastSentTime)/1e6)
			_, respTimestamp, _, err = c.api.UpdateMessage(c.Channel, ts, content...)
			if err != nil {
				return 0, err
			}
		}
	} else {
		content = buildSimpleSlackMessage(msg, strings.Join(c.Mentions, ", "))
	}

	if respTimestamp == "" {
		_, respTimestamp, err = c.api.PostMessage(c.Channel, content...)
		if err != nil {
			return 0, err
		}
	}

	unixTs, err = strconv.ParseFloat(respTimestamp, 64)

	return int64(unixTs * 1e6), nil
}

func buildSimpleSlackMessage(msg *alertMsg, mentions string) []slack.MsgOption {
	prefix := "🔴 Warning: "
	if msg.resolved {
		prefix = "🟢 Resolved: "
	}

	msgs := []slack.Block{
		slack.SectionBlock{
			Type: slack.MBTSection,
			Fields: []*slack.TextBlockObject{
				{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("%s %s\n%s", prefix, msg.node, mentions),
				},
			},
		},
		slack.SectionBlock{
			Type: slack.MBTSection,
			Fields: []*slack.TextBlockObject{
				{
					Type: slack.MarkdownType,
					Text: msg.message,
				},
			},
		},
	}

	return []slack.MsgOption{slack.MsgOptionBlocks(msgs...)}
}

func buildInteractiveSlackMessage(msg *alertMsg, mentions string) []slack.MsgOption {
	prefix := "🔴"
	if msg.resolved {
		prefix = "🟢"
	}

	msgs := []slack.Block{
		slack.SectionBlock{
			Type: "section",
			Text: &slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("*%s %s*", prefix, msg.node),
			},
		},
		slack.SectionBlock{
			Type: "section",
			Fields: []*slack.TextBlockObject{
				{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("service: %s\n"+
						"alert: %s\n\n"+
						"%s", msg.alertEvent, msg.alertLevel, msg.message),
				},
			},
		},
		slack.SectionBlock{
			Type: "section",
			Fields: []*slack.TextBlockObject{
				{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf("subscriber: %s\n", mentions),
				},
			},
		},
	}

	if !msg.resolved {
		msgs = append(msgs, slack.ActionBlock{
			Type: "actions",
			Elements: &slack.BlockElements{
				ElementSet: []slack.BlockElement{
					&slack.ButtonBlockElement{
						Type: "button",
						Text: &slack.TextBlockObject{
							Type:  slack.PlainTextType,
							Emoji: true,
							Text:  slkNoteActionId,
						},
						Value: slkNoteActionId,
					},
					&slack.ButtonBlockElement{
						Type: "button",
						Text: &slack.TextBlockObject{
							Type:  slack.PlainTextType,
							Emoji: true,
							Text:  slkAckActionId,
						},
						Value: slkAckActionId,
					},
					newPredefinedHoursSelectBlockElement(),
				},
			},
		})
	}
	msgs = append(msgs, slack.DividerBlock{
		Type: "divider",
	})

	if msg.resolved {
		msgs = append(msgs, slack.NewContextBlock("", slack.TextBlockObject{
			Type:  slack.MarkdownType,
			Text:  fmt.Sprintf("%s periodic checking success", slkLargeGreenCircleEmoticon),
			Emoji: false,
		}))
	}

	return []slack.MsgOption{
		slack.MsgOptionBlocks(msgs...),
	}
}

type alarmName string

func NewAlarmName(alarmer string, alertEvent alertEvent) alarmName {
	return alarmName(fmt.Sprintf("%s_%s", alarmer, alertEvent))
}

func (a alarmName) GetAlertEvent() alertEvent {
	return alertEvent(a[strings.LastIndex(string(a), "_")+1:])
}

type alertEvent string

var (
	TendermintAlarmEvent          alertEvent = "tendermint"
	TendermintStuckAlarmEvent     alertEvent = TendermintAlarmEvent + ":stuck"
	TendermintPeerAlarmEvent      alertEvent = TendermintAlarmEvent + ":peer"
	TendermintCommitAlarmEvent    alertEvent = TendermintAlarmEvent + ":commit"
	TendermintHeartbeatAlarmEvent alertEvent = TendermintAlarmEvent + ":heartbeat"

	EthAlarmEvent          alertEvent = "eth"
	EthStuckAlarmEvent     alertEvent = EthAlarmEvent + ":stuck"
	EthBehindAlarmEvent    alertEvent = EthAlarmEvent + ":behind"
	EthHeartbeatAlarmEvent alertEvent = EthAlarmEvent + ":heartbeat"
)

type alertMsg struct {
	node           string
	strategyTarget StrategyTarget

	lastSentTime int64

	alertEvent alertEvent
	alertLevel AlertLevel
	resolved   bool
	message    string
	uniqueId   string

	alarmers []Alarmer
}

type notifyDest string

const (
	pd  notifyDest = "pagerduty"
	tg  notifyDest = "telegram"
	di  notifyDest = "discord"
	slk notifyDest = "slack"
)

func (n *NodeInfo) alert(rbr *repository.Repository, st StrategyTarget, message string, alertLevel AlertLevel, ae alertEvent, resolved bool, id *string) {
	uniq := n.Name
	if id != nil {
		uniq = *id
	}

	alarmers := n.Alarmer.getAlarmers()
	a := &alertMsg{
		node:           n.Name,
		strategyTarget: st,
		resolved:       resolved,
		alertLevel:     alertLevel,
		message:        message,
		uniqueId:       uniq,
		alertEvent:     ae,
		alarmers:       alarmers,
	}

	if n.strategyAlertStatuses == nil {
		n.strategyAlertStatuses = make(map[StrategyTarget]*StrategyAlertStatus)
	}

	if n.strategyAlertStatuses[st] == nil {
		n.strategyAlertStatuses[st] = new(StrategyAlertStatus)
	}

	// if it's resolving message, check agentMark is there.
	if !a.resolved {

		agentMarks, err := rbr.FindAgentMarkByAgentNameAndTime(n.Name, time.Now())

		// there is set to not alert.
		if err == nil && len(agentMarks) > 0 {
			return
		}
	}

	n.notifyAlerts(a)

	n.strategyAlertStatuses[st].alertMtx.Lock()
	defer n.strategyAlertStatuses[st].alertMtx.Unlock()

	if n.strategyAlertStatuses[st].activeAlert == nil {
		n.strategyAlertStatuses[st].activeAlert = make(map[alertEvent]time.Time)
	}

	if resolved && !n.strategyAlertStatuses[st].activeAlert[ae].IsZero() {
		delete(n.strategyAlertStatuses[st].activeAlert, ae)
		return
	} else if resolved {
		return
	}

	n.strategyAlertStatuses[st].activeAlert[ae] = time.Now()

}

func (n *NodeInfo) notifyAlerts(msg *alertMsg) {
	var (
		err error
		wg  sync.WaitGroup
	)
	for _, alarmer := range msg.alarmers {
		wg.Add(1)
		go func(alm Alarmer) {
			defer wg.Done()

			// fill alarmCache if it's empty
			if n.strategyAlertStatuses[msg.strategyTarget].alarmCache == nil {
				n.strategyAlertStatuses[msg.strategyTarget].alarmCache = make(map[alarmName]int64)
			}

			// if msg.resolved == true, and also it's exist on alarmCache, delete it.
			if msg.resolved {
				if n.strategyAlertStatuses[msg.strategyTarget].alarmCache[NewAlarmName(alm.getName(), msg.alertEvent)] != 0 {
					ts := n.strategyAlertStatuses[msg.strategyTarget].alarmCache[NewAlarmName(alm.getName(), msg.alertEvent)]
					msg.lastSentTime = ts
					n.strategyAlertStatuses[msg.strategyTarget].alertMtx.Lock()
					delete(n.strategyAlertStatuses[msg.strategyTarget].alarmCache, NewAlarmName(alm.getName(), msg.alertEvent))
					n.strategyAlertStatuses[msg.strategyTarget].alertMtx.Unlock()
				} else {
					// if there is no alarmCache but received resolved msg, it'll be dropped.
					return
				}
			}

			// if current alarmer is not allocated with this alertLevel, skip it.
			if !slices.Contains(alm.getTargetAlertLevels(), msg.alertLevel) {
				n.logger.Debug(fmt.Sprintf("skipping alarmer %v: %v", alm.getName(), msg.alertLevel))
				return
			}

			// if this alarmer is set to not send resolvMsg, return.
			if msg.resolved && !alarmer.shouldSendResolveMsg() {
				n.logger.Debug(fmt.Sprintf("skipping to send resolve message %v: %v", alm.getName(), msg.alertLevel))
				return
			}

			// if current msg != resolved, and it's not qualified to resend, return.
			if lastSentTime, ok := n.strategyAlertStatuses[msg.strategyTarget].alarmCache[NewAlarmName(alm.getName(), msg.alertEvent)]; !msg.resolved && ok {
				if time.UnixMicro(lastSentTime).Add(alm.getResendDuration()).After(time.Now()) {
					n.logger.Info(fmt.Sprintf("skipping to resend message %v: %v. lastSentTime: %v, resendDuration: %v", alm.getName(), msg.alertEvent, lastSentTime, alm.getResendDuration()))
					return
				}
			}

			var (
				unixTs int64
			)
			unixTs, err = alm.notify(msg)
			if err != nil {
				n.logger.Error(fmt.Sprintf("error notifying alert %v: %v", alm.getName(), err))
				return
			}

			// if it's not resolved msg(== new alarm), set to alarmCache.
			if !msg.resolved {
				n.strategyAlertStatuses[msg.strategyTarget].alertMtx.Lock()
				if unixTs == 0 {
					n.logger.Warningf("no unix timestamp received. please check it")
					unixTs = time.Now().Unix() * 1e6
				}

				n.strategyAlertStatuses[msg.strategyTarget].alarmCache[NewAlarmName(alm.getName(), msg.alertEvent)] = unixTs
				n.strategyAlertStatuses[msg.strategyTarget].alertMtx.Unlock()
			}

		}(alarmer)
	}
	wg.Wait()
}

func (n *NodeInfo) saveAlerts(wrepo repository.Repository, commitId string, resolveTS time.Time) error {

	// get only not resolved alertRecords
	ntRslvAlerts, err := wrepo.FindAlertRecordsByInstanceAndResolvTimestamp(n.Name, nil)
	if err != nil {
		return err
	}

	storedAlarms, err := wrepo.FindActiveAlarmsByNodeName(n.Name)
	if err != nil {
		return errors.New("error loading alert records" + err.Error())
	}

	var (
		currentNtResolvAlertIds []string
	)
	for st, alertStatus := range n.strategyAlertStatuses {
		var newNotResolvedAlerts []repository.StoreEntity
		for alrtEvent, t := range alertStatus.activeAlert {
			alertRecord, err := repository.NewAlertRecord(&t, nil, string(st), string(alrtEvent), n.Name, commitId)
			if err != nil {
				return errors.New("error creating alert record" + err.Error())
			}

			var updtNotResolvAlert *repository.AlertRecord
			// if there is already notResolvedAlert, just update timestamp.
			for _, nra := range ntRslvAlerts {
				if nra.Instance == alertRecord.Instance &&
					nra.Target == alertRecord.Target &&
					nra.AlertEvent == alertRecord.AlertEvent {

					updtNotResolvAlert = &nra
					currentNtResolvAlertIds = append(currentNtResolvAlertIds, nra.AlertRecordUUID)
					break
				}
			}
			if updtNotResolvAlert == nil {
				newNotResolvedAlerts = append(newNotResolvedAlerts, *alertRecord)
			}
		}
		err = wrepo.SaveAll(newNotResolvedAlerts)
		if err != nil {
			return errors.New("error saving alert record: " + err.Error())
		}

		var newActiveAlarms []repository.StoreEntity
		for alrm, t := range alertStatus.alarmCache {
			activeAlrm, err := repository.NewActiveAlarm(t, string(alrm), string(st), n.Name, commitId)
			if err != nil {
				return errors.New("error creating active alarm" + err.Error())
			}

			var alreadyStoredAlrm *repository.ActiveAlarm
			// if there is already notResolvedAlert, just update timestamp.
			for _, storedAlarm := range storedAlarms {
				if storedAlarm.Instance == n.Name &&
					storedAlarm.Target == string(st) &&
					storedAlarm.AlarmerName == string(alrm) {

					alreadyStoredAlrm = &storedAlarm
					alreadyStoredAlrm.SentTime = t
					break
				}
			}
			if alreadyStoredAlrm == nil {
				newActiveAlarms = append(newActiveAlarms, *activeAlrm)
			} else {
				err = wrepo.UpdateActiveAlarmSentTime(*alreadyStoredAlrm, alreadyStoredAlrm.SentTime)
				if err != nil {
					return errors.New("error saving alert record" + err.Error())
				}
			}
		}

		err = wrepo.SaveAll(newActiveAlarms)
		if err != nil {
			return errors.New("error saving alert record" + err.Error())
		}

	}

	for _, nra := range ntRslvAlerts {
		if !slices.Contains(currentNtResolvAlertIds, nra.AlertRecordUUID) {
			err = wrepo.UpdateResolvTs(nra, resolveTS)
			if err != nil {
				return errors.New("error updating alert record" + err.Error())
			}
		}
	}

	var toDeleteAlrms []repository.ActiveAlarm
	for _, storedAlarm := range storedAlarms {
		// not exists
		if _, ok := n.strategyAlertStatuses[StrategyTarget(storedAlarm.Target)].alarmCache[alarmName(storedAlarm.AlarmerName)]; !ok {
			toDeleteAlrms = append(toDeleteAlrms, storedAlarm)
		}
	}

	err = wrepo.DeleteActiveAlarms(toDeleteAlrms)
	if err != nil {
		return errors.New("error deleting alert records" + err.Error())
	}

	return nil
}

func (n *NodeInfo) loadAlerts(repo *repository.Repository) error {
	ntRslvAlerts, err := repo.FindAlertRecordsByInstanceAndResolvTimestamp(n.Name, nil)
	if err != nil {
		return errors.New("error loading alert records" + err.Error())
	}

	var alertStatuses = make(map[StrategyTarget]map[alertEvent]time.Time)

	for _, nra := range ntRslvAlerts {
		if alertStatuses[StrategyTarget(nra.Target)] == nil {
			alertStatuses[StrategyTarget(nra.Target)] = make(map[alertEvent]time.Time)
		}
		alertStatuses[StrategyTarget(nra.Target)][alertEvent(nra.AlertEvent)] = *nra.StartTimestamp
	}

	alrms, err := repo.FindActiveAlarmsByNodeName(n.Name)
	if err != nil {
		return errors.New("error loading alert records" + err.Error())
	}

	var alrmStatuses = make(map[StrategyTarget]map[alarmName]int64)

	for _, alrm := range alrms {
		if alrmStatuses[StrategyTarget(alrm.Target)] == nil {
			alrmStatuses[StrategyTarget(alrm.Target)] = make(map[alarmName]int64)
		}
		alrmStatuses[StrategyTarget(alrm.Target)][alarmName(alrm.AlarmerName)] = alrm.SentTime
	}

	if n.strategyAlertStatuses == nil {
		n.strategyAlertStatuses = make(map[StrategyTarget]*StrategyAlertStatus)
	}
	for target, as := range alertStatuses {
		if n.strategyAlertStatuses[target] == nil {
			n.strategyAlertStatuses[target] = new(StrategyAlertStatus)
		}
		n.strategyAlertStatuses[target].activeAlert = as
	}

	for target, as := range alrmStatuses {
		if n.strategyAlertStatuses[target] == nil {
			n.strategyAlertStatuses[target] = new(StrategyAlertStatus)
		}
		n.strategyAlertStatuses[target].alarmCache = as
	}

	for target, _ := range n.strategyAlertStatuses {
		n.strategyAlertStatuses[target].alertMtx = sync.Mutex{}
	}

	return nil
}
