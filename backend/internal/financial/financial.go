package financial

import "time"

type AgingBucket string

const (
	Current    AgingBucket = "current"
	Days1To30  AgingBucket = "1-30"
	Days31To60 AgingBucket = "31-60"
	Days61Plus AgingBucket = "61+"
)

func BucketFor(dueDate, asOf time.Time) AgingBucket {
	due := dateOnly(dueDate)
	now := dateOnly(asOf)
	days := int(now.Sub(due).Hours() / 24)
	switch {
	case days <= 0:
		return Current
	case days <= 30:
		return Days1To30
	case days <= 60:
		return Days31To60
	default:
		return Days61Plus
	}
}

func VarianceMinor(budgetMinor, actualMinor int64) int64 {
	return budgetMinor - actualMinor
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
