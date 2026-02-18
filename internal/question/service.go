package question

import "context"

type Service struct{}

type Question struct {
	ID           int64
	QuestionType string
	StemHTML     string
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Question, error) {
	_ = ctx
	_ = id
	return nil, nil
}
