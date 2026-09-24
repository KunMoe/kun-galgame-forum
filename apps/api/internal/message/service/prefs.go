package service

import "strings"

const KeyChat = "chat"

var LocalNotificationTypes = []string{
	string(NotifyUpvoted), string(NotifyLiked), string(NotifyFavorite),
	string(NotifyReplied), string(NotifyCommented), string(NotifyMentioned),
	string(NotifyFollowed),
	string(NotifySolution), string(NotifyPinReply), string(NotifyExpired),
	string(NotifyRequested), string(NotifyMerged), string(NotifyDeclined),
	string(NotifyLotteryWon), string(NotifyLotteryClosed),
	string(NotifyLotteryExpired), string(NotifyPollClosed),
	"quiz-answered",
}

func SplitMuted(muted []string) (local []string, chatMuted bool) {
	for _, k := range muted {
		switch {
		case k == KeyChat:
			chatMuted = true
		case strings.HasPrefix(k, "wiki:"):
		default:
			local = append(local, k)
		}
	}
	return local, chatMuted
}
