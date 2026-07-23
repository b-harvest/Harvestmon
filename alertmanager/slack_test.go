package main

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/slack-go/slack"
)

// Regression guard for the bug where the alertEvent was read from
// blockSet[1].Fields, which is always nil for messages built by
// alertMessageBlocks (the text lives in blockSet[1].Text instead) — so the
// parsing loop in the Slack "Start" handler never actually matched anything.
func TestAlertEventFromMessageBlocks_MatchesAlertMessageBlocks(t *testing.T) {
	msg := &AlertMessage{
		Instance:   "node-1",
		AlertEvent: "disk-full",
		AlertLevel: "critical",
		Summary:    "disk usage above 90%",
		Status:     model.AlertFiring,
	}

	blocks := alertMessageBlocks(msg, "@oncall")

	got := alertEventFromMessageBlocks(blocks)
	if got != string(msg.AlertEvent) {
		t.Fatalf("alertEventFromMessageBlocks() = %q, want %q", got, msg.AlertEvent)
	}
}

func TestAlertEventFromMessageBlocks_NoMatch(t *testing.T) {
	cases := map[string][]slack.Block{
		"empty": {},
		"too short": {
			slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "*node-1*", false, false), nil, nil),
		},
		"wrong block type": {
			slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "*node-1*", false, false), nil, nil),
			slack.NewDividerBlock(),
		},
		"missing service prefix": {
			slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "*node-1*", false, false), nil, nil),
			slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "no alert event here", false, false), nil, nil),
		},
	}

	for name, blocks := range cases {
		t.Run(name, func(t *testing.T) {
			if got := alertEventFromMessageBlocks(blocks); got != "" {
				t.Fatalf("alertEventFromMessageBlocks() = %q, want empty string", got)
			}
		})
	}
}
