package saga

import (
	"context"
	"errors"
)

var (
	errDuplicateState     = errors.New("saga: duplicate state")
	errInvalidStateChange = errors.New("saga: invalid state change")
	errNoAggregator       = errors.New("saga: no aggregator found")
)

type smImpl struct {
	aggMap       map[State]Aggregator
	validChanges map[State]map[State]bool
}

func (s *smImpl) run(ctx context.Context, tx Transaction) (Transaction, error) {
	agg := s.aggMap[tx.State()]
	if agg == nil {
		return nil, errNoAggregator
	}

	nextTx, err := agg.Execute(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !s.validChanges[tx.State()][nextTx.State()] {
		return nil, errInvalidStateChange
	}

	return nextTx, nil
}

func (s *smImpl) addState(state State, agg Aggregator, validNextStates ...State) error {
	if s.aggMap[state] != nil {
		return errDuplicateState
	}

	s.aggMap[state] = agg
	s.validChanges[state] = map[State]bool{}
	for _, validState := range validNextStates {
		s.validChanges[state][validState] = true
	}

	return nil
}
