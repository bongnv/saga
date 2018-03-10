package saga

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmImpl_addState(t *testing.T) {
	sm := &smImpl{
		aggMap:       map[State]Aggregator{},
		validChanges: map[State]map[State]bool{},
	}

	err1 := sm.addState(State(1), &MockAggregator{})
	assert.Nil(t, err1, "should be success")
	err2 := sm.addState(State(1), &MockAggregator{})
	_ = assert.NotNil(t, err2, "should be an error") &&
		assert.True(t, strings.Contains(err2.Error(), "duplicate"), "should contain duplicate message")
}
