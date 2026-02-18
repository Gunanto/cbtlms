package report

import "context"

type Service struct{}

type ExamSummary struct {
	ExamID         int64
	Participants   int
	AverageScore   float64
	HighestScore   float64
	LowestScore    float64
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) SummaryByExam(ctx context.Context, examID int64) (*ExamSummary, error) {
	_ = ctx
	_ = examID
	return nil, nil
}
