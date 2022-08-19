package ut

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var (
	logger = log.New(os.Stderr, "st:", log.Lmsgprefix)
)

type timeItem struct {
	Name string
	Now  time.Time
}

type FuncCalcTime struct {
	Name      string
	id        string
	Stage     []timeItem
	LinePrint bool
	start     time.Time
}

// Print i为截止下标
func (c *FuncCalcTime) Print(endIndexs ...int) {
	var endIndex = len(c.Stage)
	if len(endIndexs) > 0 {
		endIndex = endIndexs[0]
	}
	if endIndex < 1 {
		return
	}
	var end = endIndex

	if endIndex > len(c.Stage) {
		end = len(c.Stage)
	}

	var st strings.Builder
	var preTime time.Time
	for i, item := range c.Stage {
		if i > end {
			break
		}

		var nowTime = preTime
		if i == 0 {
			nowTime = c.start
		}
		st.WriteString(fmt.Sprintf(" -->%s:%s ", item.Name, item.Now.Sub(nowTime)))

		preTime = item.Now
	}

	logger.Printf("[%s][%s][%s]%s", c.Name, c.id, c.start.Format("2006-01-02 15:04:05"), st.String())
}

func (c *FuncCalcTime) Add(t string, msg ...interface{}) {
	name := t
	if len(msg) >= 1 {
		name = fmt.Sprintf(t, msg...)
	}
	var item = timeItem{
		Name: name,
		Now:  time.Now(),
	}
	c.Stage = append(c.Stage, item)
	if c.LinePrint {
		c.Print()
	}
}
func (c *FuncCalcTime) ChangeLinePrint(show bool) {
	c.LinePrint = show
}

// NewFST 方法计时器
func NewFST(name string) FuncCalcTime {
	c := FuncCalcTime{
		Name:  name,
		Stage: make([]timeItem, 0),
		start: time.Now(),
		id:    RandomStr(12),
	}
	return c
}
