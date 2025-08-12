package logger

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"sync"
	"time"
)

// TimeSizeRotator is an io.Writer that rotates logs by both time and size.
type TimeSizeRotator struct {
	mu           sync.Mutex
	filename     string
	maxSize      int
	maxBackups   int
	maxAge       int
	compress     bool
	timeUnit     TimeUnit
	lw           *lumberjack.Logger
}

// NewTimeSizeRotator creates a new TimeSizeRotator.
func NewTimeSizeRotator(filename string, maxSize, maxBackups, maxAge int, compress bool, timeUnit TimeUnit) io.Writer {
	r := &TimeSizeRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
		timeUnit:   timeUnit,
	}

	// Initialize the first logger
	r.rotate(time.Now())

	// Start a goroutine to rotate the file based on the time unit
	go r.rotationManager()

	return r
}

// Write implements io.Writer.
func (r *TimeSizeRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lw.Write(p)
}

func (r *TimeSizeRotator) rotationManager() {
	for {
		now := time.Now()
		next := r.nextRotationTime(now)
		timer := time.NewTimer(next.Sub(now))
		<-timer.C
		r.rotate(time.Now())
	}
}

func (r *TimeSizeRotator) nextRotationTime(now time.Time) time.Time {
	switch r.timeUnit {
	case Hour:
		return now.Truncate(time.Hour).Add(time.Hour)
	default: // Day or other units
		// Truncate to the beginning of the current day and add 24 hours to get to the next day
		year, month, day := now.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
	}
}

func (r *TimeSizeRotator) rotate(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get the current date string from the time unit's format
	dateStr := now.Format(r.timeUnit.goFormat())

	// Form the new filename
	newFilename := fmt.Sprintf("%s.%s.log", r.filename, dateStr)

	// If the filename is the same, no need to rotate
	if r.lw != nil && r.lw.Filename == newFilename {
		return
	}

	// Close the old logger if it exists
	if r.lw != nil {
		r.lw.Close()
	}

	r.lw = &lumberjack.Logger{
		Filename:   newFilename,
		MaxSize:    r.maxSize,
		MaxBackups: r.maxBackups,
		MaxAge:     r.maxAge,
		Compress:   r.compress,
	}
}
