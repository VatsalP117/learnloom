package httpapp

import (
	"context"
	"sync"

	"github.com/VatsalP117/learnloom/internal/domain"
)

type SourceValidator func(
	context.Context,
	domain.SourceDefinition,
) ([]domain.SourceItem, []string, error)

var sourceValidators sync.Map

func (s *Server) SetSourceValidator(validator SourceValidator) {
	if validator == nil {
		sourceValidators.Delete(s)
		return
	}
	sourceValidators.Store(s, validator)
}

func (s *Server) sourceValidator() SourceValidator {
	value, ok := sourceValidators.Load(s)
	if !ok {
		return nil
	}
	validator, _ := value.(SourceValidator)
	return validator
}
