package constants

var TopicSectionConsume = map[string]bool{
	"g-seeking": true,
	"g-other":   true,
	"t-help":    true,
}

const (
	// A lottery hands out scarce goods, so the entry bar is higher than the
	// poll's: either an account old enough to be inconvenient to farm, or enough
	// moemoepoint that the account has actually contributed. Moderators bypass
	// both via lottery.create_any.
	LotteryMinAccountAgeDays = 30
	LotteryMinMoemoepoint    = 100

	MaxLotteriesPerTopic = 10

	MaxSlotsPerPrize = 500

	MaxPollsPerTopic = 30
)
