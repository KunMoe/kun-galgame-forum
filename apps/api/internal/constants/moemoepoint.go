package constants

const (
	RewardCreateTopic    = 3
	RewardCreateGalgame  = 3
	RewardCreateResource = 3
	RewardCreateToolset  = 3
	RewardReply          = 1
	RewardPRMerge        = 1

	CostConsumeSection = 10
	// Charged by the OAuth account center (infra setting auth.name_change_cost), not here.
	CostChangeUsername = 17
	CostUpvoteSender   = 10
	RewardUpvoteOwner  = 5

	RatingRewardHigh         = 10
	RatingRewardMedium       = 5
	RatingRewardLow          = 3
	RatingLenThresholdHigh   = 666
	RatingLenThresholdMedium = 233

	QuizCreateReward = 2

	RewardBestAnswer = 7
	CheckinMaxReward = 7

	// A balance buys posting room: dailyLimit = moemoepoint/this + 1.
	DailyTopicPerMoemoepoint = 10

	TextPreviewLength = 233
)
