package session

// FatigueDrop compares physical metrics between the first and second half
// of a structured session. Returned on-read (not persisted).
type FatigueDrop struct {
	FirstHalf  HalfMetrics `json:"first_half"`
	SecondHalf HalfMetrics `json:"second_half"`
	// DropPercentage is the second-half decline in distance covered per sample,
	// as a fraction (0 = no drop). Clamped at 0 when the 2nd half was stronger.
	DropPercentage float64 `json:"drop_percentage"`
	// Percentage change from 1st to 2nd half (negative = drop).
	HRAvgPctChange      *float64 `json:"hr_avg_pct_change"`
	HighIntensityPctChange *float64 `json:"high_intensity_pct_change"`
}

// HalfMetrics aggregates samples for one half.
type HalfMetrics struct {
	DistanceM       float64  `json:"distance_m"`
	HRAvg           *float64 `json:"hr_avg"`
	SpeedMaxKMH     *float64 `json:"speed_max_kmh"`
	HighIntensityS  int      `json:"high_intensity_s"`
	SampleCount     int      `json:"sample_count"`
}

const minSamplesPerHalf = 6

// highIntensityHRThreshold returns the HR value above which intensity is "high"
// (85 % of HRmax).
func highIntensityHRThreshold(hrMax int) int {
	return int(float64(hrMax) * 0.85)
}

// ComputeFatigueDrop returns per-half metrics for a structured session. Returns
// nil when the session is not structured, has no halftime offset, or does not
// have enough samples in both halves.
func ComputeFatigueDrop(mode string, samples []Sample, halftimeOffsetS *int, hrMax int) *FatigueDrop {
	if mode != ModeStructured || halftimeOffsetS == nil {
		return nil
	}
	if len(samples) < minSamplesPerHalf*2 {
		return nil
	}

	var firstHalf, secondHalf []Sample
	ht := *halftimeOffsetS
	for _, s := range samples {
		if s.Half != nil {
			if *s.Half == 1 {
				firstHalf = append(firstHalf, s)
			} else {
				secondHalf = append(secondHalf, s)
			}
		} else if s.TOffsetS < ht {
			firstHalf = append(firstHalf, s)
		} else {
			secondHalf = append(secondHalf, s)
		}
	}

	if len(firstHalf) < minSamplesPerHalf || len(secondHalf) < minSamplesPerHalf {
		return nil
	}

	threshold := highIntensityHRThreshold(hrMax)
	first := computeHalfMetrics(firstHalf, threshold)
	second := computeHalfMetrics(secondHalf, threshold)

	fd := FatigueDrop{
		FirstHalf:  first,
		SecondHalf: second,
	}
	fd.HRAvgPctChange = pctChange(first.HRAvg, second.HRAvg)
	fd.HighIntensityPctChange = pctChangeInt(first.HighIntensityS, second.HighIntensityS)
	fd.DropPercentage = distanceRateDrop(first, second)

	return &fd
}

// distanceRateDrop returns the fraction by which distance-per-sample fell in the
// second half (0 when the 2nd half held up or improved).
func distanceRateDrop(first, second HalfMetrics) float64 {
	if first.SampleCount == 0 || second.SampleCount == 0 {
		return 0
	}
	rate1 := first.DistanceM / float64(first.SampleCount)
	rate2 := second.DistanceM / float64(second.SampleCount)
	if rate1 <= 0 {
		return 0
	}
	drop := (rate1 - rate2) / rate1
	if drop < 0 {
		return 0
	}
	return drop
}

func computeHalfMetrics(half []Sample, highThreshold int) HalfMetrics {
	var hrSum, highCount int
	var speedMax, speedSum float64
	var speedCount int
	minOffset, maxOffset := -1, -1

	for _, s := range half {
		if s.HR != nil {
			hrSum += *s.HR
			if *s.HR >= highThreshold {
				highCount++
			}
		}
		if s.SpeedKMH != nil {
			speedSum += *s.SpeedKMH
			speedCount++
			if *s.SpeedKMH > speedMax {
				speedMax = *s.SpeedKMH
			}
		}
		if minOffset < 0 || s.TOffsetS < minOffset {
			minOffset = s.TOffsetS
		}
		if s.TOffsetS > maxOffset {
			maxOffset = s.TOffsetS
		}
	}

	count := len(half)
	var hrAvg *float64
	if hrSum > 0 {
		v := float64(hrSum) / float64(count)
		hrAvg = &v
	}
	var spdMax *float64
	if speedMax > 0 {
		spdMax = &speedMax
	}

	// Distance ≈ average speed (m/s) × half duration (s).
	var distanceM float64
	if speedCount > 0 && maxOffset > minOffset {
		avgSpeedMS := (speedSum / float64(speedCount)) / 3.6
		distanceM = avgSpeedMS * float64(maxOffset-minOffset)
	}

	// Estimate time in high intensity by using the sample count and a default
	// 10 s interval (the watch records a sample every ~10 s). This gives a
	// reasonable approximation of seconds spent in high intensity.
	highIntensityS := highCount * 10

	return HalfMetrics{
		DistanceM:       distanceM,
		HRAvg:           hrAvg,
		SpeedMaxKMH:     spdMax,
		HighIntensityS:  highIntensityS,
		SampleCount:     count,
	}
}

func pctChange(first, second *float64) *float64 {
	if first == nil || second == nil || *first == 0 {
		return nil
	}
	v := ((*second - *first) / *first) * 100
	return &v
}

func pctChangeInt(first, second int) *float64 {
	if first == 0 {
		return nil
	}
	v := (float64(second-first) / float64(first)) * 100
	return &v
}
