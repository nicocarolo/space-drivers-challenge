package travel

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Point(t *testing.T) {
	p := Point{
		Lat: -100.121091,
		Lng: 2.19918919,
	}

	assert.Equal(t, "-100.121091, 2.19918919", p.String())

	var newPoint Point
	newPoint.FromString(p.String())

	assert.Equal(t, p.Lat, newPoint.Lat)
	assert.Equal(t, p.Lng, newPoint.Lng)
}
