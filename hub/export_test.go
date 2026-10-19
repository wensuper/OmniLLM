package hub

import "time"

// SubscriptionPaceForTest exposes the per-device pace to tests.
type SubscriptionPaceForTest struct{ p subscriptionPace }

func (s *SubscriptionPaceForTest) Allow(who string, now time.Time) error { return s.p.allow(who, now) }
