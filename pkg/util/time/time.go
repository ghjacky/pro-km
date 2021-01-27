package time

import (
	"fmt"
	"math"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
)

const (
	// CSTTimeFormat is a CST Time format string
	CSTTimeFormat = "Mon 2006-01-02 15:04:05 CST"
)

// Duration return the split int
func Duration(start time.Time, end time.Time) (days, hours, minutes, seconds int) {
	if start.After(end) {
		return 0, 0, 0, 0
	}
	diff := end.Sub(start)
	days = int(diff.Hours() / 24)
	hours = int(math.Mod(diff.Hours(), 24))
	minutes = int(math.Mod(diff.Minutes(), 60))
	seconds = int(math.Mod(diff.Seconds(), 60))
	return
}

// Elapsed return a string of compute the duration between start and end
func Elapsed(start *time.Time, end *time.Time) string {
	if start == nil {
		return ""
	}
	if end == nil {
		now := time.Now()
		end = &now
	}
	elapsed := ""
	days, hours, minutes, seconds := Duration(*start, *end)
	if days > 0 {
		elapsed = fmt.Sprintf("%d天", days)
	}
	if hours > 0 {
		elapsed = fmt.Sprintf("%s%d小时", elapsed, hours)
	}
	if days == 0 {
		if minutes > 0 {
			elapsed = fmt.Sprintf("%s%d分", elapsed, minutes)
		}
	}
	if hours == 0 {
		if seconds >= 0 {
			elapsed = fmt.Sprintf("%s%d秒", elapsed, seconds)
		}
	}
	return elapsed
}

// ParseTime parse a format string to time
func ParseTime(f, t string) *time.Time {
	if t == "" {
		return nil
	}
	time, err := time.ParseInLocation(f, t, time.Local)
	if err != nil {
		alog.Errorf("Parse str %s to format %q time failed: %v", t, f, err)
		return nil
	}
	return &time
}
