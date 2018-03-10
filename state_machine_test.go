package saga

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmImpl_addState(t *testing.T) {
	sm := &smImpl{
		aggMap:       map[State]Aggregator{},
		validChanges: map[State]map[State]bool{},
	}

	mockAgg := &MockAggregator{}
	err1 := sm.addState(State(1), mockAgg, State(2))
	assert.Nil(t, err1, "should be success")
	assert.True(t, sm.validChanges[State(1)][State(2)])
	assert.EqualValues(t, sm.aggMap[State(1)], mockAgg)

	err2 := sm.addState(State(1), &MockAggregator{})
	_ = assert.NotNil(t, err2, "should be an error") &&
		assert.True(t, strings.Contains(err2.Error(), "duplicate"), "should contain duplicate message")
}

func TestSmImpl_run(t *testing.T) {
	sm := &smImpl{
		aggMap:       map[State]Aggregator{},
		validChanges: map[State]map[State]bool{},
	}
	mockAgg := &MockAggregator{}
	assert.Nil(t, sm.addState(State(1), mockAgg, State(2)))
	t.Run("no-aggregator", func(t *testing.T) {
		mockTx := &MockTransaction{}
		mockTx.On("State").Return(State(2)).Once()
		nextTx, err := sm.run(context.Background(), mockTx)
		assert.Nil(t, nextTx)
		assert.Equal(t, errNoAggregator, err)
		mockAgg.AssertExpectations(t)
	})

	t.Run("err-aggregator", func(t *testing.T) {
		mockTx := &MockTransaction{}
		mockTx.On("State").Return(State(1)).Once()
		mockAgg.On("Execute", context.Background(), mockTx).Return(nil, errors.New("random error")).Once()
		nextTx, err := sm.run(context.Background(), mockTx)
		assert.Nil(t, nextTx)
		_ = assert.NotNil(t, err) &&
			assert.True(t, strings.Contains(err.Error(), "random error"))
		mockAgg.AssertExpectations(t)
	})

	t.Run("invalid-state-change", func(t *testing.T) {
		mockTx := &MockTransaction{}
		mockTx2 := &MockTransaction{}
		mockTx.On("State").Return(State(1)).Twice()
		mockTx2.On("State").Return(State(3)).Once()
		mockAgg.On("Execute", context.Background(), mockTx).Return(mockTx2, nil).Once()
		nextTx, err := sm.run(context.Background(), mockTx)
		assert.Nil(t, nextTx)
		assert.Equal(t, errInvalidStateChange, err)
		mockAgg.AssertExpectations(t)
		mockTx.AssertExpectations(t)
		mockTx2.AssertExpectations(t)
	})

	t.Run("happy-path", func(t *testing.T) {
		mockTx := &MockTransaction{}
		mockTx2 := &MockTransaction{}
		mockTx.On("State").Return(State(1)).Twice()
		mockTx2.On("State").Return(State(2)).Once()
		mockAgg.On("Execute", context.Background(), mockTx).Return(mockTx2, nil).Once()
		nextTx, err := sm.run(context.Background(), mockTx)
		assert.EqualValues(t, mockTx2, nextTx)
		assert.Nil(t, err)
		mockAgg.AssertExpectations(t)
		mockTx.AssertExpectations(t)
		mockTx2.AssertExpectations(t)
	})
}
