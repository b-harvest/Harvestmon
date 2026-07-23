package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	"github.com/b-harvest/Harvestmon/repository"
	"github.com/prometheus/common/model"
	"github.com/slack-go/slack"
	"golang.org/x/exp/slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ----------------- TYPES & CONSTANTS -------------------

// AlertLevel is an alias type for string to represent the severity/level of the alert.
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

// AlarmerConfig is a struct for multiple platform alarm configurations.
type AlarmerConfig struct {
	// PagerDuty configurations
	Pagerdutys []PDConfig `toml:"pagerdutys"`
	// Slack configurations
	Slacks []SlackConfig `toml:"slacks"`
}

func (c *AlarmerConfig) getAlarmers() []Alarmer {
	var alarmers []Alarmer
	for _, pdConf := range c.Pagerdutys {
		// Only append if Enabled is true
		if pdConf.Enabled {
			alarmers = append(alarmers, &pdConf)
		}
	}
	for _, slkConf := range c.Slacks {
		// Only append if Enabled is true
		if slkConf.Enabled {
			alarmers = append(alarmers, &slkConf)
		}
	}
	return alarmers
}

// Alarmer is an interface that platform-specific alarmers implement.
type Alarmer interface {
	getName() string
	getTargetAlertLevels() []AlertLevel
	getResendDuration() time.Duration
	shouldSendResolveMsg() bool
	notify(msg *AlertMessage) (sentMicro int64, err error)
}

// PDConfig is the information required to send alerts to PagerDuty.
type PDConfig struct {
	Enabled           bool          `toml:"enabled"`
	ApiKey            string        `toml:"apiKey"`
	DefaultSeverity   string        `toml:"defaultSeverity"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`
}

func (c *PDConfig) getName() string                    { return string(pd) }
func (c *PDConfig) getTargetAlertLevels() []AlertLevel { return c.TargetAlertLevels }
func (c *PDConfig) getResendDuration() time.Duration   { return c.ResendDuration }
func (c *PDConfig) shouldSendResolveMsg() bool         { return c.ResolveMsg }

// pdDedupKey builds the PagerDuty dedup key for an instance+alertEvent pair.
// It must stay identical across trigger/acknowledge/resolve calls for the same
// alert so they all target the same PagerDuty incident. Keying on instance
// alone (the previous behavior) collapsed distinct alert events on the same
// instance into one incident, so resolving one closed them all.
func pdDedupKey(instance InstanceName, ae alertEvent) string {
	return fmt.Sprintf("%s_%s", instance, ae)
}

// manageEvent sends a PagerDuty Events API v2 action (trigger/acknowledge/resolve)
// for the incident identified by instance+alertEvent.
func (c *PDConfig) manageEvent(action string, instance InstanceName, ae alertEvent, payload *pagerduty.V2Payload) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: c.ApiKey,
		Action:     action,
		DedupKey:   pdDedupKey(instance, ae),
		Payload:    payload,
	})
	return err
}

// notify sends alert messages to PagerDuty via API v2.
func (c *PDConfig) notify(msg *AlertMessage) (int64, error) {
	action := "trigger"
	if msg.Status == model.AlertResolved {
		action = "resolve"
	}

	err := c.manageEvent(action, msg.Instance, msg.AlertEvent, &pagerduty.V2Payload{
		Summary:  fmt.Sprintf("[%s] %s", msg.Instance, msg.Summary),
		Source:   string(msg.Instance),
		Severity: c.DefaultSeverity,
	})
	return time.Now().UnixMicro(), err
}

// acknowledge tells PagerDuty the incident for instance+alertEvent has been
// acknowledged (e.g. via the Slack "Ack" button) without resolving it.
func (c *PDConfig) acknowledge(instance InstanceName, ae alertEvent) error {
	return c.manageEvent("acknowledge", instance, ae, nil)
}

// resolve tells PagerDuty the incident for instance+alertEvent has been
// manually resolved (e.g. via the Slack "Start" button, which re-arms alerting).
func (c *PDConfig) resolve(instance InstanceName, ae alertEvent) error {
	return c.manageEvent("resolve", instance, ae, nil)
}

// SlackConfig holds the information needed to publish to a Slack channel for sending alerts.
type SlackConfig struct {
	Enabled           bool          `toml:"enabled"`
	BotToken          string        `toml:"botToken"`
	Channel           string        `toml:"channel"`
	Mentions          []string      `toml:"mentions"`
	TargetAlertLevels []AlertLevel  `toml:"targetAlertLevels"`
	ResolveMsg        bool          `toml:"resolveMsg"`
	ResendDuration    time.Duration `toml:"resendDuration"`

	VerificationToken string `toml:"verificationToken"`
	SigningSecret     string `toml:"signingSecret"`

	api *slack.Client
}

func (c *SlackConfig) getName() string                    { return string(slk) }
func (c *SlackConfig) getTargetAlertLevels() []AlertLevel { return c.TargetAlertLevels }
func (c *SlackConfig) getResendDuration() time.Duration   { return c.ResendDuration }
func (c *SlackConfig) shouldSendResolveMsg() bool         { return c.ResolveMsg }

// notify sends alert messages to Slack channel.
func (c *SlackConfig) notify(msg *AlertMessage) (int64, error) {
	if c.api == nil {
		c.api = slack.New(c.BotToken)
	}

	content := buildInteractiveSlackMessage(msg, strings.Join(c.Mentions, ", "))
	var (
		err           error
		respTimestamp string
	)

	// If this is a resolved alert and we have a lastSentTime, update the original Slack message.
	if msg.Status == model.AlertResolved && msg.lastSentTime != 0 {
		ts := fmt.Sprintf("%f", float64(msg.lastSentTime)/1e6)
		_, respTimestamp, _, err = c.api.UpdateMessage(c.Channel, ts, content...)
		if err != nil {
			return 0, err
		}
	}

	// If we didn't get back a timestamp from UpdateMessage (i.e. the message was never posted),
	// then do a fresh PostMessage.
	if respTimestamp == "" {
		_, respTimestamp, err = c.api.PostMessage(c.Channel, content...)
		if err != nil {
			return 0, err
		}
	}

	unixTs, parseErr := strconv.ParseFloat(respTimestamp, 64)
	if parseErr != nil {
		// Fallback if Slack returns something unexpected
		return time.Now().UnixMicro(), nil
	}
	return int64(unixTs * 1e6), nil
}

// buildInteractiveSlackMessage returns a Slack message block set with interactive buttons, etc.
func buildInteractiveSlackMessage(msg *AlertMessage, mentions string) []slack.MsgOption {
	return []slack.MsgOption{
		slack.MsgOptionBlocks(alertMessageBlocks(msg, mentions)...),
	}
}

// alertMessageBlocks builds the block set for an alert message. Kept separate from
// buildInteractiveSlackMessage so alertEventFromMessageBlocks (slack.go) can be
// tested against the exact block shape it needs to parse.
func alertMessageBlocks(msg *AlertMessage, mentions string) []slack.Block {
	prefix := "🔴" // Red by default
	if msg.Status == model.AlertResolved {
		prefix = "🟢" // Green if resolved
	}

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("*%s %s*", prefix, msg.Instance),
				false,
				false,
			),
			nil, nil),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf(
					"service: %s\nalert: %s\n\n%s",
					msg.AlertEvent,
					msg.AlertLevel,
					msg.Summary,
				),
				false,
				false,
			),
			nil, nil),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("subscriber: %s\n", mentions),
				false,
				false,
			),
			nil, nil),
	}

	// Only show actions if not resolved
	if msg.Status != model.AlertResolved {
		var blockElements []slack.BlockElement
		buttonIds := []string{slkNoteActionId, slkAckActionId}
		for _, buttonID := range buttonIds {
			blockElements = append(blockElements,
				slack.NewButtonBlockElement(
					buttonID,
					buttonID,
					slack.NewTextBlockObject(
						slack.PlainTextType,
						buttonID,
						true,
						false,
					),
				))
		}
		// Potentially add other action elements here...

		// Example: A dropdown to pick hours
		blockElements = append(blockElements, predefinedHoursSelectBlockElement())

		blocks = append(blocks, slack.ActionBlock{
			Type: "actions",
			Elements: &slack.BlockElements{
				ElementSet: blockElements,
			},
		})
	}

	// Divider
	blocks = append(blocks, slack.DividerBlock{
		Type: slack.MBTDivider,
	})

	// If resolved, show a context block
	if msg.Status == model.AlertResolved {
		blocks = append(blocks, slack.NewContextBlock(
			"",
			slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("%s periodic checking success", slkLargeGreenCircleEmoticon),
			},
		))
	}

	return blocks
}

// alarmName is a string that encodes the Alarmer + alertEvent
type alarmName string

// NewAlarmName constructs an alarmName from alarmer + alertEvent.
func NewAlarmName(alarmer string, alertEvent alertEvent) alarmName {
	return alarmName(fmt.Sprintf("%s_%s", alarmer, alertEvent))
}

func (a alarmName) GetAlertEvent() alertEvent {
	str := string(a)
	idx := strings.LastIndex(str, "_")
	if idx < 0 {
		return alertEvent(str)
	}
	return alertEvent(str[idx+1:])
}

type alertEvent string
type notifyDest string

const (
	pd  notifyDest = "pagerduty"
	slk notifyDest = "slack"
)

// AlertMessage captures the relevant pieces of an alert event.
type AlertMessage struct {
	Instance InstanceName      `json:"instance"`
	Target   TargetName        `json:"target"`
	StartsAt string            `json:"startsAt"`
	Summary  string            `json:"summary"`
	Status   model.AlertStatus `json:"status"`

	AlertEvent alertEvent `json:"alertEvent"`
	AlertLevel AlertLevel `json:"alertLevel"`

	Labels map[string]string `json:"labels"`

	// Internal fields (not in JSON)
	lastSentTime int64
	alarmers     []Alarmer
}

type AlertMessages struct {
	Alerts []AlertMessage `json:"alerts"`
}

// alert is called when a new alert arrives from external system (e.g., Prometheus).
func (am *AlertManager) alert(a AlertMessage) {
	// Assign alarmers from config
	a.alarmers = am.AlarmerConfig.getAlarmers()

	// Initialize am.record if needed
	if am.record == nil {
		am.record = make(map[InstanceName]map[TargetName]*Record)
	}

	if am.record[a.Instance] == nil {
		am.record[a.Instance] = make(map[TargetName]*Record)
	}

	// Make sure we have a lock for this instance/target
	if am.record[a.Instance][a.Target] == nil {
		am.record[a.Instance][a.Target] = &Record{
			mtx:         &sync.Mutex{},
			activeAlert: make(map[alertEvent]time.Time),
			alarmCache:  make(map[alarmName]int64),
		}
	}

	now := time.Now()
	agentMarks, err := am.reader.FindAgentMarkByInstanceAndTime(string(a.Instance), &now)
	if err != nil {
		am.logger.Errorf("failed to find marks for alert %s: %s", a.Instance, err)
		return
	}

	// If we are receiving an active (firing) alert, check for agent marks
	// to potentially skip alerts that are "silenced" or manually suppressed
	if a.Status != model.AlertResolved && len(agentMarks) != 0 {
		am.logger.Debugf("marks for alert %s exist, skipping alert...", a.Instance)
		return
	} else if a.Status == model.AlertResolved && len(agentMarks) != 0 {
		var shouldDeleteMarks []repository.StoreEntity
		for _, mark := range agentMarks {
			mark.MarkEnd = &now
			shouldDeleteMarks = append(shouldDeleteMarks, mark)
		}
		err = am.writer.SaveAll(shouldDeleteMarks)
		if err != nil {
			am.logger.Errorf("failed to resolv marks for alert %s: %s", a.Instance, err)
		}
	}

	// Send notifications to alarmers
	am.notifyAlerts(&a)

	// Update local active-alert state
	rec := am.record[a.Instance][a.Target]
	rec.mtx.Lock()
	defer rec.mtx.Unlock()

	if a.Status == model.AlertResolved {
		// If resolved, remove from activeAlert
		if !rec.activeAlert[a.AlertEvent].IsZero() {
			delete(rec.activeAlert, a.AlertEvent)
		}
		return
	}

	// If alert is firing, mark as active
	rec.activeAlert[a.AlertEvent] = time.Now()
}

// notifyAlerts sends the alert to the relevant alarmers in parallel.
func (am *AlertManager) notifyAlerts(msg *AlertMessage) {
	rec := am.record[msg.Instance][msg.Target]
	if rec == nil {
		return
	}

	// We'll do concurrency, but be mindful of rec usage
	rec.mtx.Lock()
	if rec.alarmCache == nil {
		rec.alarmCache = make(map[alarmName]int64)
	}
	rec.mtx.Unlock()

	var wg sync.WaitGroup

	for _, alarmer := range msg.alarmers {
		wg.Add(1)
		go func(alm Alarmer) {
			defer wg.Done()

			// Quick checks on supported alert-level and "resolve" messages
			if !slices.Contains(alm.getTargetAlertLevels(), msg.AlertLevel) {
				am.logger.Debugf("Skipping alarmer %v: alert level %v not in targets (event: %v)",
					alm.getName(), msg.AlertLevel, msg.AlertEvent)
				return
			}
			if msg.Status == model.AlertResolved && !alm.shouldSendResolveMsg() {
				am.logger.Debugf("Skipping resolve message to %v for %v (resolve not configured)",
					alm.getName(), msg.AlertEvent)
				return
			}

			alarmKey := NewAlarmName(alm.getName(), msg.AlertEvent)
			rec.mtx.Lock()
			lastSent, alreadyCached := rec.alarmCache[alarmKey]
			rec.mtx.Unlock()

			// If this is a resolve, but there's no record in alarmCache, nothing to update
			if msg.Status == model.AlertResolved && !alreadyCached {
				// This means we never sent a "fire" for this alarmer, so skip
				return
			}

			// If resolved, set msg.lastSentTime to the old one (for Slack update)
			if msg.Status == model.AlertResolved && alreadyCached {
				msg.lastSentTime = lastSent
			}

			// If not resolved, check if we are too early to re-send
			if msg.Status != model.AlertResolved && alreadyCached {
				resendAfter := time.UnixMicro(lastSent).Add(alm.getResendDuration())
				if time.Now().Before(resendAfter) {
					am.logger.Infof("Skipping re-send for %v (event: %v). Last sent: %v. Resend duration: %v",
						alm.getName(), msg.AlertEvent, time.UnixMicro(lastSent), alm.getResendDuration())
					return
				}
			}

			// Actually notify
			sentTimestamp, err := alm.notify(msg)
			if err != nil {
				am.logger.Errorf("Error notifying alert to %v: %v", alm.getName(), err)
				return
			}

			// If it's a new or re-sent alarm, store the new timestamp in alarmCache
			rec.mtx.Lock()
			defer rec.mtx.Unlock()

			if msg.Status == model.AlertResolved {
				// Once resolved is sent, remove from alarmCache
				delete(rec.alarmCache, alarmKey)
			} else {
				// If we didn't get a valid sentTimestamp, fall back
				if sentTimestamp == 0 {
					am.logger.Warningf("No valid timestamp from %s; using time.Now()", alm.getName())
					sentTimestamp = time.Now().UnixMicro()
				}
				rec.alarmCache[alarmKey] = sentTimestamp
			}

			am.logger.Infof("updated alert: %v, before: %v, after: %v", msg.Summary, time.UnixMicro(lastSent), time.UnixMicro(sentTimestamp))
		}(alarmer)
	}

	wg.Wait()
}

// saveAlerts persists the current alert state to some data store (via am.writer).
// 'resolveTS' is the timestamp used to mark older alerts as resolved.
func (am *AlertManager) saveAlerts(resolveTS time.Time) error {
	// find all not-resolved (active) alerts in the DB
	notResolvedAlrts, err := am.reader.FindAlertRecordsByResolvTimestamp(nil)
	if err != nil {
		return fmt.Errorf("error loading not resolved alerts: %w", err)
	}

	// find all active alarms in the DB
	storedAlarms, err := am.writer.FindActiveAlarms()
	if err != nil {
		return fmt.Errorf("error loading active alarms: %w", err)
	}

	for instance, instanceStat := range am.record {
		for target, targetStat := range instanceStat {
			targetStat.mtx.Lock()
			for alrtEvent, t := range targetStat.activeAlert {

				var updateCandidate *repository.AlertRecord
				newAlert, err := repository.NewAlertRecord(&t, nil, string(target), string(alrtEvent), string(instance))
				if err != nil {
					am.logger.Warningf("failed to create new active alarm (%v)", err)
					continue
				}

				for idx, nra := range notResolvedAlrts {
					if nra.Instance == newAlert.Instance &&
						nra.Target == newAlert.Target &&
						nra.AlertEvent == newAlert.AlertEvent {
						updateCandidate = &nra
						notResolvedAlrts = append(notResolvedAlrts[:idx], notResolvedAlrts[idx+1:]...)
						break
					}
				}

				if updateCandidate == nil {
					if err = am.writer.Save(newAlert); err != nil {
						am.logger.Warningf("failed to save new active alert (%v)", err)
					}
				}
			}

			for _, nra := range notResolvedAlrts {
				err = am.writer.UpdateResolvTs(nra, resolveTS)
				if err != nil {
					am.logger.Warningf("error updating alert record (%s) as resolved: %v", nra.AlertRecordUUID, err)
				}
			}

			for alrm, t := range targetStat.alarmCache {

				var storedAlrmPtr *repository.ActiveAlarm

				newAlrm, err := repository.NewActiveAlarm(t, string(alrm), string(target), string(instance))
				if err != nil {
					am.logger.Warningf("failed to create new active alarm (%v)", err)
					continue
				}

				for idx, storedAlarm := range storedAlarms {
					if storedAlarm.Instance == newAlrm.Instance &&
						storedAlarm.Target == newAlrm.Target &&
						storedAlarm.AlarmerName == newAlrm.AlarmerName {

						storedAlrmPtr = &storedAlarm
						storedAlarms = append(storedAlarms[:idx], storedAlarms[idx+1:]...)
						break
					}
				}

				if storedAlrmPtr == nil {
					err = am.writer.Save(newAlrm)
					if err != nil {
						am.logger.Warningf("error saving alert record (%s) as resolved: %v", alrm, err)
					}
				} else if storedAlrmPtr.SentTime != newAlrm.SentTime {
					am.logger.Infof("update alarm(%v) sentTime(%v)", newAlrm, time.UnixMicro(newAlrm.SentTime))
					if upErr := am.writer.UpdateActiveAlarmSentTime(*newAlrm, newAlrm.SentTime); upErr != nil {
						am.logger.Warningf("error updating active alarm sent time: %v", upErr)
					}
				}
			}

			targetStat.mtx.Unlock()
		}
	}

	if err := am.writer.DeleteActiveAlarms(storedAlarms); err != nil {
		am.logger.Warningf("error deleting new active alarms: %v", err)
	}

	am.logger.Info("alert records saved (including any deletions of stale records)")
	return nil
}

// loadAlerts initializes am.record from what we find in the DB.
func (am *AlertManager) loadAlerts() error {
	if am.record == nil {
		am.record = make(map[InstanceName]map[TargetName]*Record)
	}

	// 1) Load non-resolved alerts
	ntRslvAlerts, err := am.reader.FindAlertRecordsByResolvTimestamp(nil)
	if err != nil {
		return fmt.Errorf("error loading not-resolved alerts: %w", err)
	}
	for _, nra := range ntRslvAlerts {
		instance := InstanceName(nra.Instance)
		target := TargetName(nra.Target)
		ae := alertEvent(nra.AlertEvent)

		if am.record[instance] == nil {
			am.record[instance] = make(map[TargetName]*Record)
		}
		if am.record[instance][target] == nil {
			am.record[instance][target] = &Record{
				mtx:         &sync.Mutex{},
				activeAlert: make(map[alertEvent]time.Time),
				alarmCache:  make(map[alarmName]int64),
			}
		}

		am.record[instance][target].activeAlert[ae] = *nra.StartTimestamp
	}

	// 2) Load active alarms
	alrms, err := am.reader.FindActiveAlarms()
	if err != nil {
		return fmt.Errorf("error loading active alarms: %w", err)
	}
	for _, alrm := range alrms {
		instance := InstanceName(alrm.Instance)
		target := TargetName(alrm.Target)
		aName := alarmName(alrm.AlarmerName)

		if am.record[instance] == nil {
			am.record[instance] = make(map[TargetName]*Record)
		}
		if am.record[instance][target] == nil {
			am.record[instance][target] = &Record{
				mtx:         &sync.Mutex{},
				activeAlert: make(map[alertEvent]time.Time),
				alarmCache:  make(map[alarmName]int64),
			}
		}

		am.record[instance][target].alarmCache[aName] = alrm.SentTime
	}

	return nil
}
