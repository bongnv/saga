package saga_test

import (
	"context"
	"fmt"

	"github.com/bongnv/saga"
)

const (
	stateInit saga.State = iota
	stateBookedHotel
	stateCanceledHotel
	stateBookHotelFailed
	stateBookedFlight
	stateCanceledFlight
	stateBookFlightFailed
)

type exampleTransaction struct {
	state saga.State
}

func (t *exampleTransaction) State() saga.State {
	return t.state
}

type aggBookHotel struct{}

func (a *aggBookHotel) Execute(ctx context.Context, tx saga.Transaction) (saga.Transaction, error) {
	fmt.Println("Book hotel")
	return &exampleTransaction{
		state: stateBookedHotel,
	}, nil
}

type aggBookFlight struct {
	nextState saga.State
}

func (a *aggBookFlight) Execute(ctx context.Context, tx saga.Transaction) (saga.Transaction, error) {
	fmt.Println("Book flight")
	return &exampleTransaction{
		state: a.nextState,
	}, nil
}

type aggCancelHotel struct{}

func (a *aggCancelHotel) Execute(ctx context.Context, tx saga.Transaction) (saga.Transaction, error) {
	fmt.Println("Cancel hotel")
	return &exampleTransaction{
		state: stateCanceledHotel,
	}, nil
}

func ExampleExecutor() {
	exampleConfig := saga.Config{
		InitState: stateInit,
		Activities: []saga.Activity{
			{
				Aggregator:      &aggBookHotel{},
				SuccessState:    stateBookedHotel,
				FailureState:    stateBookHotelFailed,
				RolledBackState: stateCanceledHotel,
			},
			{
				Aggregator: &aggBookFlight{
					nextState: stateBookedFlight,
				},
				SuccessState:    stateBookedFlight,
				FailureState:    stateBookFlightFailed,
				RolledBackState: stateCanceledFlight,
			},
		},
	}

	executor, err := saga.NewExecutor(exampleConfig)
	if err != nil {
		fmt.Printf("Failed to create saga executor, err: %v.\n", err)
		return
	}

	finalTx, err := executor.Execute(context.Background(), &exampleTransaction{
		state: stateInit,
	})
	if err != nil {
		fmt.Printf("Failed to execute saga transaction, err: %v.\n", err)
		return
	}
	if finalTx.State() == stateBookedFlight {
		fmt.Println("OK")
	}
	// Output:
	// Book hotel
	// Book flight
	// OK
}
func ExampleExecutor_rollback() {
	exampleConfig := saga.Config{
		InitState: stateInit,
		Activities: []saga.Activity{
			{
				Aggregator:      &aggBookHotel{},
				SuccessState:    stateBookedHotel,
				FailureState:    stateBookHotelFailed,
				Compensation:    &aggCancelHotel{},
				RolledBackState: stateCanceledHotel,
			},
			{
				Aggregator: &aggBookFlight{
					nextState: stateBookFlightFailed,
				},
				SuccessState:    stateBookedFlight,
				FailureState:    stateBookFlightFailed,
				RolledBackState: stateCanceledFlight,
			},
		},
	}

	executor, err := saga.NewExecutor(exampleConfig)
	if err != nil {
		fmt.Printf("Failed to create saga executor, err: %v.\n", err)
		return
	}

	finalTx, err := executor.Execute(context.Background(), &exampleTransaction{
		state: stateInit,
	})
	if err != nil {
		fmt.Printf("Failed to execute saga transaction, err: %v.\n", err)
		return
	}
	if finalTx.State() == stateCanceledHotel {
		fmt.Println("OK")
	}
	// Output:
	// Book hotel
	// Book flight
	// Cancel hotel
	// OK
}
