package apiv1

import (
	"context"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listWorkCollectedMonths(_ context.Context, in *listWorkCollectedMonthsInput) (*collectedMonthsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	fn := s.collectedMonths
	if fn == nil && s.lists != nil {
		fn = s.lists.ListCollectedCalendar
	}
	if fn == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	// The strip takes the reader's SFW gate: a month that can only answer with
	// rows this reader may not see is a filter option that looks broken.
	rows, err := fn(!in.IncludeNSFW)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]WorkCollectedMonth, len(rows))
	for i, r := range rows {
		items[i] = WorkCollectedMonth{Year: r.Year, Month: r.Month}
	}
	if items == nil {
		items = []WorkCollectedMonth{}
	}
	return &collectedMonthsOutput{Body: WorkCollectedMonths{Object: "work_collected_months", Items: items}}, nil
}

type collectedMonthsOutput struct {
	Body WorkCollectedMonths
}

func (s *Service) WithCollectedMonths(fn func(bool) ([]repository.CollectedMonth, error)) *Service {
	s.collectedMonths = fn
	return s
}
