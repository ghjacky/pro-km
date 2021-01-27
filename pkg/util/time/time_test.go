package time

import (
	"fmt"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	tests := []struct {
		start time.Time
		end   time.Time
		exp   []int
		elp   string
	}{
		{
			start: time.Date(2020, 4, 13, 11, 0, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 12, 0, 0, 0, time.Local),
			exp:   []int{0, 1, 0, 0},
			elp:   "1小时",
		},
		{
			start: time.Date(2020, 4, 12, 11, 0, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 12, 0, 0, 0, time.Local),
			exp:   []int{1, 1, 0, 0},
			elp:   "1天1小时",
		},
		{
			start: time.Date(2020, 4, 10, 12, 50, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 8, 10, 0, 0, time.Local),
			exp:   []int{2, 19, 20, 0},
			elp:   "2天19小时",
		},
		{
			start: time.Date(2020, 4, 12, 12, 50, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 8, 10, 10, 0, time.Local),
			exp:   []int{0, 19, 20, 10},
			elp:   "19小时20分",
		},
		{
			start: time.Date(2020, 4, 13, 8, 20, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 8, 50, 10, 0, time.Local),
			exp:   []int{0, 0, 30, 10},
			elp:   "30分10秒",
		},
		{
			start: time.Date(2020, 4, 14, 11, 0, 0, 0, time.Local),
			end:   time.Date(2020, 4, 13, 12, 0, 0, 0, time.Local),
			exp:   []int{0, 0, 0, 0},
			elp:   "",
		},
	}

	for _, test := range tests {
		ds, hours, minutes, seconds := Duration(test.start, test.end)
		if ds != test.exp[0] || hours != test.exp[1] || minutes != test.exp[2] || seconds != test.exp[3] {
			t.Errorf("Not right: %v != %v", []int{ds, hours, minutes, seconds}, test.exp)
		}

		elapsed := Elapsed(&test.start, &test.end)
		if elapsed != test.elp {
			t.Errorf("Elapsed not right: %s != %s", elapsed, test.elp)
		}
	}
}

func TestParseTime(t *testing.T) {
	d := time.Date(2020, 3, 21, 1, 10, 51, 0, time.Local)
	tests := []struct {
		form string
		date string
		time *time.Time
	}{
		{
			form: "Mon 2006-01-02 15:04:05 CST",
			date: "Sat 2020-03-21 01:10:51 CST",
			time: &d,
		},
		{
			form: "Mon 2006-01-02 15:04:05 CST",
			date: "",
			time: nil,
		},
	}

	for _, test := range tests {
		time := ParseTime(test.form, test.date)
		fmt.Printf("time: %v\n", time)
		if time == nil && test.time != nil {
			t.Errorf("time not equals: %v != %v", time, test.time)
		}
		if time != nil && !time.Equal(*test.time) {
			t.Errorf("time not equals: %v != %v", time, test.time)
		}
	}
}

func TestElapsed(t *testing.T) {
	form := "Mon 2006-01-02 15:04:05 CST"
	tests := []struct {
		date string
	}{
		{
			date: "Fri 2020-04-24 18:00:13 CST",
		},
		{
			date: "Mon 2019-11-25 23:05:35 CST",
		},
	}
	for _, test := range tests {
		start := ParseTime(form, test.date)
		now := time.Now()
		elap := Elapsed(start, &now)
		fmt.Printf("elap = %s\n", elap)
	}
}
