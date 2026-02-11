package monitor

import (
	"sync"
	"time"
)

// RateDetector tracks event rates and detects spikes using a sliding window.
type RateDetector struct {
	mu         sync.Mutex
	window     time.Duration
	buckets    []int64     // per-second counters
	timestamps []time.Time // timestamp for each bucket
	threshold  float64     // spike threshold multiplier (e.g., 3.0 = 3x average)
}

// NewRateDetector creates a rate detector with the given window duration and spike threshold.
// threshold is the multiplier over the moving average that triggers a spike alert.
// E.g., threshold=3.0 means alert when current rate exceeds 3x the average.
func NewRateDetector(window time.Duration, threshold float64) *RateDetector {
	if window < time.Second {
		window = 10 * time.Second
	}
	if threshold <= 0 {
		threshold = 3.0
	}
	return &RateDetector{
		window:    window,
		threshold: threshold,
	}
}

// Record adds an event at the current time.
// Returns true if a spike is detected.
func (r *RateDetector) Record() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.prune(now)

	// Find or create bucket for current second.
	truncated := now.Truncate(time.Second)
	if len(r.timestamps) > 0 && r.timestamps[len(r.timestamps)-1].Equal(truncated) {
		r.buckets[len(r.buckets)-1]++
	} else {
		r.buckets = append(r.buckets, 1)
		r.timestamps = append(r.timestamps, truncated)
	}

	return r.isSpiking()
}

// CurrentRate returns events per second over the last window.
func (r *RateDetector) CurrentRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.prune(time.Now())

	if len(r.buckets) == 0 {
		return 0
	}

	var total int64
	for _, b := range r.buckets {
		total += b
	}
	seconds := r.window.Seconds()
	if seconds == 0 {
		return 0
	}
	return float64(total) / seconds
}

// LatestSecondRate returns the event count in the most recent second.
func (r *RateDetector) LatestSecondRate() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.buckets) == 0 {
		return 0
	}

	now := time.Now().Truncate(time.Second)
	last := r.timestamps[len(r.timestamps)-1]
	if now.Equal(last) {
		return r.buckets[len(r.buckets)-1]
	}
	return 0
}

// prune removes buckets older than the window. Must be called with lock held.
func (r *RateDetector) prune(now time.Time) {
	cutoff := now.Add(-r.window)
	i := 0
	for i < len(r.timestamps) && r.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		r.buckets = r.buckets[i:]
		r.timestamps = r.timestamps[i:]
	}
}

// isSpiking checks if the latest bucket exceeds threshold * average. Must be called with lock held.
func (r *RateDetector) isSpiking() bool {
	if len(r.buckets) < 3 {
		return false // not enough data
	}

	// Average of all but the last bucket.
	var sum int64
	for i := 0; i < len(r.buckets)-1; i++ {
		sum += r.buckets[i]
	}
	avg := float64(sum) / float64(len(r.buckets)-1)
	if avg == 0 {
		return false
	}

	latest := float64(r.buckets[len(r.buckets)-1])
	return latest > avg*r.threshold
}
