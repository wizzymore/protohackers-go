package traffic_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wizzymore/tcp-go/traffic"
)

func TestBalancerCanRoundRobin(t *testing.T) {
	balancer := traffic.NewBalancer[int]()
	assert.Equal(t, 0, balancer.Index())
	{
		_, ok := balancer.Get()
		assert.False(t, ok)
	}
	balancer.Add(1)
	balancer.Add(2)
	balancer.Add(3)
	val, ok := balancer.Get()
	assert.True(t, ok)
	assert.Equal(t, val, 1)
	val, ok = balancer.Get()
	assert.True(t, ok)
	assert.Equal(t, val, 2)
	val, ok = balancer.Get()
	assert.True(t, ok)
	assert.Equal(t, val, 3)
	val, ok = balancer.Get()
	assert.True(t, ok)
	assert.Equal(t, val, 1)
}

func TestBalancerHandlesRemovesOk(t *testing.T) {
	balancer := traffic.NewBalancer[int]()
	balancer.Add(1)
	balancer.Add(2)
	balancer.Add(3)

	_, _ = balancer.Get()
	_, _ = balancer.Get()
	// Next is 3, self.index should be 2
	// removing 3 should move us back to the front
	balancer.Remove(3)
	val, ok := balancer.Get()
	assert.True(t, ok)
	assert.Equal(t, 1, val)
}
