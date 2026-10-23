package gate

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/trustclient"
)

const (
	SubjectKindTopic = "forum_topic"
	SubjectKindReply = "forum_reply"

	SubjectKindTopicComment      = "forum_comment"
	SubjectKindTopicPoll         = "forum_topic_poll"
	SubjectKindTopicLottery      = "forum_topic_lottery"
	SubjectKindGalgameRating     = "galgame_rating"
	SubjectKindGalgameResource   = "galgame_resource"
	SubjectKindGalgameCollection = "galgame_collection"
	SubjectKindGalgameQuiz       = "galgame_quiz"
	SubjectKindGalgameQuizAnswer = "galgame_quiz_answer"
	SubjectKindToolset           = "galgame_toolset"
	SubjectKindToolsetResource   = "galgame_toolset_resource"
)

const scanTimeout = 5 * time.Second

type Scanner interface {
	Scan(ctx context.Context, req trustclient.ScanRequest) (*trustclient.ScanResult, error)
}

type ScanService struct {
	sc Scanner
}

func NewScanService(sc Scanner) *ScanService {
	return &ScanService{sc: sc}
}

func (s *ScanService) Enabled() bool { return s != nil && s.sc != nil }

func (s *ScanService) ScanBg(subjectKind, subjectID, text string, authorID int64, checkSkipped bool) {
	if !s.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
		defer cancel()
		req := trustclient.ScanRequest{SubjectKind: subjectKind, SubjectID: subjectID, Text: text, CheckSkipped: checkSkipped}
		if authorID > 0 {
			req.AuthorID = &authorID
		}
		if _, err := s.sc.Scan(ctx, req); err != nil {
			slog.Warn("trust scan", "subject_kind", subjectKind, "subject_id", subjectID, "err", err)
		}
	}()
}
