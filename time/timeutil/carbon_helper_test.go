package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCarbonDateUtilities(t *testing.T) {
	now := time.Date(2025, 4, 15, 21, 18, 15, 0, time.UTC)

	t.Run("AddDays", func(t *testing.T) {
		expected := time.Date(2025, 4, 25, 21, 18, 15, 0, time.UTC)
		actual := now.AddDate(0, 0, 10)
		assert.Equal(t, expected, actual)
	})

	t.Run("SubDays", func(t *testing.T) {
		expected := time.Date(2025, 4, 5, 21, 18, 15, 0, time.UTC)
		actual := now.AddDate(0, 0, -10)
		assert.Equal(t, expected, actual)
	})

	t.Run("IsWeekend", func(t *testing.T) {
		weekday := now.Weekday()
		isWeekend := weekday == time.Saturday || weekday == time.Sunday
		assert.False(t, isWeekend)
	})

	t.Run("StartOfWeek", func(t *testing.T) {
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday()-time.Monday))
		expected := time.Date(2025, 4, 14, 21, 18, 15, 0, time.UTC)
		assert.Equal(t, expected, startOfWeek)
	})

	t.Run("EndOfWeek", func(t *testing.T) {
		util := FromTime(now)
		endOfWeek := util.EndOfWeek().ToTime()
		expected := time.Date(2025, 4, 20, 21, 18, 15, 0, time.UTC)

		assert.Equal(t, expected.Year(), endOfWeek.Year())
		assert.Equal(t, expected.Month(), endOfWeek.Month())
		assert.Equal(t, expected.Day(), endOfWeek.Day())

	})

	t.Run("DiffInDays", func(t *testing.T) {
		future := now.AddDate(0, 0, 10)
		diff := int(future.Sub(now).Hours() / 24)
		assert.Equal(t, 10, diff)
	})
}
