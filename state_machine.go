package saga

import (
	"context"
	"errors"
)

type smImpl struct {
	aggMap       map[State]Aggregator
	validChanges map[State]map[State]bool
}

func (s *smImpl) run(ctx context.Context, tx Transaction) (Transaction, error) {
	agg := s.aggMap[tx.State()]
	if agg == nil {
		return nil, errors.New("saga: no aggregator found")
	}

	nextTx, err := agg.Execute(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !s.validChanges[tx.State()][nextTx.State()] {
		return nil, errors.New("saga: invalid state change")
	}

	return nextTx, nil
}

func (s *smImpl) addState(state State, agg Aggregator, validNextStates ...State) error {
	if s.aggMap[state] != nil {
		return errors.New("saga: duplicate state")
	}

	s.aggMap[state] = agg
	s.validChanges[state] = map[State]bool{}
	for _, validState := range validNextStates {
		s.validChanges[state][validState] = true
	}

	return nil
}
