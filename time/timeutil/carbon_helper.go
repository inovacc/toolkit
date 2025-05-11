package timeutil

import (
	"time"

	"gitee.com/dromara/carbon/v2"
)

// Initialize default configuration for Carbon library.
func init() {
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.UTC,
		Locale:       "en",
		WeekStartsAt: carbon.Monday,
	})
}

// TimeUtil wraps a Carbon instance for chainable time utility operations.
type TimeUtil struct {
	value *carbon.Carbon
}

// New returns a new TimeUtil initialized to the current time.
func New() *TimeUtil {
	return &TimeUtil{value: carbon.Now()}
}

// Parse parses a date string into a TimeUtil instance.
// Returns an error if the string is not a valid date.
func Parse(dateStr string) (*TimeUtil, error) {
	c := carbon.Parse(dateStr)
	if c.Error != nil {
		return nil, c.Error
	}
	return &TimeUtil{value: c}, nil
}

// FromTime creates a TimeUtil from a standard time.Time value.
func FromTime(t time.Time) *TimeUtil {
	return &TimeUtil{value: carbon.CreateFromStdTime(t)}
}

// AddDays adds the specified number of days and returns the updated TimeUtil.
func (t *TimeUtil) AddDays(days int) *TimeUtil {
	t.value = t.value.AddDays(days)
	return t
}

// SubtractDays subtracts the specified number of days and returns the updated TimeUtil.
func (t *TimeUtil) SubtractDays(days int) *TimeUtil {
	t.value = t.value.SubDays(days)
	return t
}

// Format formats the internal time using the given layout string.
func (t *TimeUtil) Format(layout string) string {
	return t.value.Layout(layout)
}

// DifferenceInDays returns the number of days between this and another TimeUtil.
func (t *TimeUtil) DifferenceInDays(other *TimeUtil) int64 {
	return t.value.DiffInDays(other.value)
}

// IsWeekend returns true if the current time is a weekend.
func (t *TimeUtil) IsWeekend() bool {
	return t.value.IsWeekend()
}

// StartOfWeek returns a new TimeUtil set to the beginning of the week.
func (t *TimeUtil) StartOfWeek() *TimeUtil {
	return &TimeUtil{value: t.value.StartOfWeek()}
}

// EndOfWeek returns a new TimeUtil set to the end of the week.
func (t *TimeUtil) EndOfWeek() *TimeUtil {
	return &TimeUtil{value: t.value.EndOfWeek()}
}

// ToTime returns the standard time.Time representation of the internal Carbon time.
func (t *TimeUtil) ToTime() time.Time {
	return t.value.StdTime()
}

// Carbon returns the underlying Carbon pointer instance.
func (t *TimeUtil) Carbon() *carbon.Carbon {
	return t.value
}
