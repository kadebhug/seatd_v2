package analytics

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidServicePeriod = errors.New("invalid service period")

type ServicePeriodWindow struct {
	Date  time.Time
	Start time.Time
	End   time.Time
}

func ServicePeriodWindowForDate(
	date time.Time,
	location *time.Location,
	daysOfWeek []int16,
	startTime pgtype.Time,
	endTime pgtype.Time,
) (ServicePeriodWindow, bool, error) {
	if location == nil {
		return ServicePeriodWindow{}, false, ErrInvalidServicePeriod
	}
	if !startTime.Valid || !endTime.Valid {
		return ServicePeriodWindow{}, false, ErrInvalidServicePeriod
	}

	localDate := time.Date(date.In(location).Year(), date.In(location).Month(), date.In(location).Day(), 0, 0, 0, 0, location)
	if !containsDay(daysOfWeek, int16(localDate.Weekday())) {
		return ServicePeriodWindow{}, false, nil
	}

	startHour, startMinute, startSecond, startNano := timeParts(startTime)
	endHour, endMinute, endSecond, endNano := timeParts(endTime)
	start := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), startHour, startMinute, startSecond, startNano, location)
	end := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), endHour, endMinute, endSecond, endNano, location)
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return ServicePeriodWindow{Date: localDate, Start: start.UTC(), End: end.UTC()}, true, nil
}

func ParseClock(value string) (pgtype.Time, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		parsed, err = time.Parse("15:04:05", value)
	}
	if err != nil {
		return pgtype.Time{}, fmt.Errorf("%w: time must be HH:MM or HH:MM:SS", ErrInvalidServicePeriod)
	}
	micros := int64(parsed.Hour()) * int64(time.Hour/time.Microsecond)
	micros += int64(parsed.Minute()) * int64(time.Minute/time.Microsecond)
	micros += int64(parsed.Second()) * int64(time.Second/time.Microsecond)
	return pgtype.Time{Microseconds: micros, Valid: true}, nil
}

func FormatClock(value pgtype.Time) string {
	if !value.Valid {
		return ""
	}
	hour, minute, second, _ := timeParts(value)
	if second == 0 {
		return two(hour) + ":" + two(minute)
	}
	return two(hour) + ":" + two(minute) + ":" + two(second)
}

func timeParts(value pgtype.Time) (int, int, int, int) {
	duration := time.Duration(value.Microseconds) * time.Microsecond
	hour := int(duration / time.Hour)
	duration -= time.Duration(hour) * time.Hour
	minute := int(duration / time.Minute)
	duration -= time.Duration(minute) * time.Minute
	second := int(duration / time.Second)
	duration -= time.Duration(second) * time.Second
	return hour, minute, second, int(duration)
}

func containsDay(days []int16, day int16) bool {
	for _, candidate := range days {
		if candidate == day {
			return true
		}
	}
	return false
}

func two(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
