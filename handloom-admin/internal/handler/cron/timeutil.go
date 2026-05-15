package cron

import "time"

// NextDayPickupSlotIST returns tomorrow at 09:00 IST (03:30 UTC) — the canonical
// pickup slot for Delhivery batch + on-demand manifests. Does NOT skip weekends;
// callers depend on EventBridge schedules to restrict invocation days.
func NextDayPickupSlotIST(now time.Time) time.Time {
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 3, 30, 0, 0, time.UTC)
}
