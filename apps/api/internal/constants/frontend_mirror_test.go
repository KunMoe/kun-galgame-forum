package constants

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// /point quotes these numbers to users as the site's rules, and the page is a
// Vue component: nothing connects its prose to the constants the server charges
// by. A tier rewritten here and forgotten there leaves the site advertising a
// price it does not charge, which is how "改名需要 17 个萌萌点" got written down
// while no code anywhere deducted 17.
const mirrorPath = "../../../web/app/constants/moemoepoint.ts"

func TestWebMirrorsTheMoemoepointRules(t *testing.T) {
	want := map[string]int{
		"createTopic":        RewardCreateTopic,
		"createGalgame":      RewardCreateGalgame,
		"createResource":     RewardCreateResource,
		"createToolset":      RewardCreateToolset,
		"reply":              RewardReply,
		"prMerge":            RewardPRMerge,
		"consumeSection":     CostConsumeSection,
		"upvoteSender":       CostUpvoteSender,
		"upvoteOwner":        RewardUpvoteOwner,
		"ratingHigh":         RatingRewardHigh,
		"ratingMedium":       RatingRewardMedium,
		"ratingLow":          RatingRewardLow,
		"ratingLenHigh":      RatingLenThresholdHigh,
		"ratingLenMedium":    RatingLenThresholdMedium,
		"quizCreate":         QuizCreateReward,
		"bestAnswer":         RewardBestAnswer,
		"checkinMax":         CheckinMaxReward,
		"dailyTopicPerPoint": DailyTopicPerMoemoepoint,
	}

	got := readMirror(t)
	for key, value := range want {
		mirrored, ok := got[key]
		if !ok {
			t.Errorf("%s does not carry %q — the page cannot quote a rule it has no number for", mirrorPath, key)
			continue
		}
		if mirrored != value {
			t.Errorf("%s: %s = %d, the server charges %d", mirrorPath, key, mirrored, value)
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("%s carries %q, which no constant here backs — either name the "+
				"constant or stop publishing the number", mirrorPath, key)
		}
	}
}

var mirrorEntryRe = regexp.MustCompile(`(?m)^\s*([a-zA-Z]+):\s*(\d+)`)

func readMirror(t *testing.T) map[string]int {
	t.Helper()
	raw, err := os.ReadFile(mirrorPath)
	if err != nil {
		t.Fatalf("read %s: %v — the web mirror moved; point this test at it rather "+
			"than deleting the check", mirrorPath, err)
	}
	out := map[string]int{}
	for _, m := range mirrorEntryRe.FindAllStringSubmatch(string(raw), -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("%s: %q is not a number", mirrorPath, m[2])
		}
		out[m[1]] = n
	}
	if len(out) == 0 {
		t.Fatalf("no entries parsed out of %s — it was restructured and this check "+
			"now asserts nothing", mirrorPath)
	}
	return out
}
