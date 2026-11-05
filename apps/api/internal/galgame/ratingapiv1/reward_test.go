package ratingapiv1

import (
	"strings"
	"testing"

	"kun-galgame-api/internal/constants"
)

func TestRatingRewardMeasuresTheTextNotTheMarkup(t *testing.T) {
	word := strings.Repeat("a", constants.RatingLenThresholdMedium-1)
	if got := ratingReward("**" + word + "**"); got != constants.RatingRewardLow {
		t.Errorf("markup counted toward the reward: got %d", got)
	}
	if got := ratingReward("||" + word + "a||"); got != constants.RatingRewardMedium {
		t.Errorf("a spoiler's text must count: got %d", got)
	}
}
