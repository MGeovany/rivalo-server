package session

// FatigueDrop compares physical metrics between the first and second half
// of a structured session. Returned on-read (not persisted).
type FatigueDrop struct {
	FirstHalf  HalfMetrics `json:"first_half"`
	SecondHalf HalfMetrics `json:"second_half"`
	// Percentage change from 1st to 2nd half (negative = drop).
	HRAvgPctChange      *float64 `json:"hr_avg_pct_change"`
	HighIntensityPctChange *float64 `json:"high_intensity_pct_change"`
}

// HalfMetrics aggregates samples for one half.
type HalfMetrics struct {
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

	return &fd
}

func computeHalfMetrics(half []Sample, highThreshold int) HalfMetrics {
	var hrSum, highCount int
	var speedMax float64

	for _, s := range half {
		if s.HR != nil {
			hrSum += *s.HR
			if *s.HR >= highThreshold {
				highCount++
			}
		}
		if s.SpeedKMH != nil && *s.SpeedKMH > speedMax {
			speedMax = *s.SpeedKMH
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

	// Estimate time in high intensity by using the sample count and a default
	// 10 s interval (the watch records a sample every ~10 s). This gives a
	// reasonable approximation of seconds spent in high intensity.
	highIntensityS := highCount * 10

	return HalfMetrics{
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
