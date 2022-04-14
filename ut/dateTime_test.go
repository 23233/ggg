package ut

import "testing"

func TestGetFirstDateOfWeek(t *testing.T) {
	t.Log(GetFirstDateOfWeek())
}

func TestGetFirstDateOfMonth(t *testing.T) {
	t.Log(GetFirstDateOfMonth())
}

func TestGetStartTimeOfToday(t *testing.T) {
	t.Log(GetStartTimeOfToday())
}

func TestGetEndTimeOfToday(t *testing.T) {
	t.Log(GetEndTimeOfToday())
}

func TestDateAll(t *testing.T) {
	t.Run("本周0点", TestGetFirstDateOfWeek)
	t.Run("本月0点", TestGetFirstDateOfMonth)
	t.Run("今日0点", TestGetStartTimeOfToday)
	t.Run("今日23点59分59秒", TestGetEndTimeOfToday)
}
