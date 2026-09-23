package service

import (
	"kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type ReplyService struct {
	replyRepo *repository.ReplyRepository
}

func NewReplyService(replyRepo *repository.ReplyRepository) *ReplyService {
	return &ReplyService{replyRepo: replyRepo}
}

func (s *ReplyService) ModerationRemove(replyID int) error {
	reply, err := s.replyRepo.FindByID(replyID)
	if err != nil {
		return nil
	}
	return s.replyRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.replyRepo.DeleteRepliesByIDs(tx, []int{replyID}); err != nil {
			return err
		}
		return recomputeTopicCounts(tx, reply.TopicID)
	})
}
