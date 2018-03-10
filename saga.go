package saga

import (
	"context"
	"errors"
)

// State of a saga transaction
type State int16

// Transaction is an interface of a saga transaction.
//
// State returns current state of a transaction.
type Transaction interface {
	State() State
}

// Aggregator is an interface of a agregator
// which execute a single step in a list of steps in a saga transaction.
//
// Execute process the current step of a transaction and
// returns another object of the transaction in next state.
// It returns an error when fails to process.
type Aggregator interface {
	Execute(ctx context.Context, tx Transaction) (Transaction, error)
}

// Logger is an interface that wraps Log function.
//
// Log persists a transaction normally for auditing or recovering
// It returns an error while fails to store.
type Logger interface {
	Log(ctx context.Context, tx Transaction) error
}

// Activity includes details of an activity in a saga
// - Aggregator
// - SuccessState when it is executed successfully
// - FailureState when it is failed to be executed.
// - RolledBackState when it is successfulled executed but rolled back
// - Compensataion aggregator to rollback
type Activity struct {
	SuccessState    State
	FailureState    State
	RolledBackState State
	Aggregator      Aggregator
	Compensation    Aggregator
}

// Saga includes details of a saga
// - InitState is the initial state
// - Activities list of activities in this saga in order
// - Logger is a logger to log state of a saga transaction
type Saga struct {
	InitState  State
	Activities []Activity
	Logger     Logger
}

// Executor to execute a saga transaction
type Executor struct {
	sm         *smImpl
	finalState map[State]bool
	logger     Logger
}

// Execute continue to process a saga transaction based on current state
// It returns a transaction object after it's executed sucessfully
// It returns error when failed to execute it
func (e *Executor) Execute(ctx context.Context, tx Transaction) (Transaction, error) {
	for {
		if e.finalState[tx.State()] {
			return tx, nil
		}

		nextTx, err := e.sm.run(ctx, tx)
		if err != nil {
			return tx, err
		}
		if err := e.logger.Log(ctx, nextTx); err != nil {
			return tx, err
		}
		tx = nextTx
	}
}

// NewExecutor creates a new Executor for a saga
func NewExecutor(saga Saga) (*Executor, error) {
	if len(saga.Activities) == 0 {
		return nil, errors.New("saga: no activity")
	}

	e := &Executor{
		sm: &smImpl{
			aggMap:       map[State]Aggregator{},
			validChanges: map[State]map[State]bool{},
		},
		logger: &nopLogger{},
		finalState: map[State]bool{
			saga.Activities[0].RolledBackState:                   true,
			saga.Activities[0].FailureState:                      true,
			saga.Activities[len(saga.Activities)-1].SuccessState: true,
		},
	}

	if saga.Logger != nil {
		e.logger = saga.Logger
	}

	// forward flow
	prevState := saga.InitState
	for _, a := range saga.Activities {
		if a.Aggregator == nil {
			return nil, errors.New("saga: nil aggregator")
		}
		if err := e.sm.addState(prevState, a.Aggregator, a.SuccessState, a.FailureState); err != nil {
			return nil, err
		}
	}

	// backward, rollback flow
	for i := 1; i < len(saga.Activities); i++ {
		preA := saga.Activities[i-1]
		curA := saga.Activities[i]
		if err := e.sm.addState(curA.FailureState, preA.Compensation, preA.RolledBackState); err != nil {
			return nil, err
		}
		if err := e.sm.addState(curA.RolledBackState, preA.Compensation, preA.RolledBackState); err != nil {
			return nil, err
		}
	}

	return e, nil
}

type nopLogger struct{}

func (l *nopLogger) Log(ctx context.Context, tx Transaction) error {
	return nil
}

//go:generate mockery -name=Aggregator -case=underscore --inpkg