package types

import (
	"time"
)

type TxCondition struct {
	EndTime		time.Time
}

func (c *TxCondition) Check() (b bool) {
	return c.EndTime.In(Loc).Sub(time.Now().In(Loc)) < time.Second
}

var DefaultCondition = TxCondition{
	EndTime:time.Now().In(Loc),
}