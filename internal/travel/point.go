package travel

import (
	"fmt"
	"strconv"
	"strings"
)

type Point struct {
	Lat float64 `json:"latitude" binding:"required"`
	Lng float64 `json:"longitude" binding:"required"`
}

func (p Point) String() string {
	lat := strconv.FormatFloat(p.Lat, 'g', -1, 64)
	lng := strconv.FormatFloat(p.Lng, 'g', -1, 64)

	return fmt.Sprintf("%s, %s", lat, lng)
}

func (p *Point) FromString(value string) (err error) {
	split := strings.Split(value, ", ")

	p.Lat, err = strconv.ParseFloat(split[0], 64)
	if err != nil {
		return err
	}

	p.Lng, err = strconv.ParseFloat(split[1], 64)
	if err != nil {
		return err
	}

	return nil
}
