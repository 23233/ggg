package logger

import (
	"time"
)

type TimeUnit string

const (
	Minute = "minute"
	Hour   = "hour"
	Day    = "day"
	Month  = "month"
	Year   = "year"
)

func (t TimeUnit) Format() string {
	switch t {
	case Minute:
		return ".%Y%m%d%H%M"
	case Hour:
		return ".%Y%m%d%H"
	case Day:
		return ".%Y%m%d"
	case Month:
		return ".%Y%m"
	case Year:
		return ".%Y"
	default:
		return ".%Y%m%d"
	}
}

func (t TimeUnit) RotationGap() time.Duration {
	switch t {
	case Minute:
		return time.Minute
	case Hour:
		return time.Hour
	case Day:
		return time.Hour * 24
	case Month:
		return time.Hour * 24 * 30
	case Year:
		return time.Hour * 24 * 365
	default:
		return time.Hour * 24
	}
}

func (t TimeUnit) goFormat() string {
	switch t {
	case Minute:
		return "200601021504"
	case Hour:
		return "2006010215"
	case Day:
		return "20060102"
	case Month:
		return "200601"
	case Year:
		return "2006"
	default:
		return "20060102"
	}
}
