package session

// Edwards TRIMP zones based on %HRmax.
// Zone boundaries: 50%, 60%, 70%, 80%, 90%
// Zone weights: 1, 2, 3, 4, 5
// Reference: Edwards S. "High performance training and racing" (1994)
// Validated against Banister's TRIMP: https://pubmed.ncbi.nlm.nih.gov/21904234/

var trimpZones = []struct {
	lowerPct float64
	upperPct float64
	weight   float64
}{
	{lowerPct: 0.50, upperPct: 0.60, weight: 1},
	{lowerPct: 0.60, upperPct: 0.70, weight: 2},
	{lowerPct: 0.70, upperPct: 0.80, weight: 3},
	{lowerPct: 0.80, upperPct: 0.90, weight: 4},
	{lowerPct: 0.90, upperPct: 1.00, weight: 5},
}

// Expected TRIMP range for normalization (0–100 scale).
// From sports science literature: a full 90-min match typically yields
// TRIMP 200–350 for most athletes. We set the reference max at 350.
const refMaxTRIMP = 350.0

// HRmaxByAge estimates maximum heart rate using the Tanaka formula.
// Tanaka H, Monahan KD, Seals DR (2001). "Age-predicted maximal heart rate revisited".
func HRmaxByAge(birthYear int, referenceYear int) int {
	age := referenceYear - birthYear
	if age < 10 {
		age = 10
	}
	return int(208.0 - 0.7*float64(age))
}

// CalculateMatchRating computes Edwards TRIMP from HR samples and HRmax,
// then normalizes to a 0–100 scale. Returns nil if there are no HR samples.
func CalculateMatchRating(samples []Sample, hrMax int, totalDurationS int) *float64 {
	if len(samples) == 0 || hrMax <= 0 || totalDurationS <= 0 {
		return nil
	}

	// Count seconds spent in each zone.
	zoneSeconds := make([]float64, 5)
	for i := range samples {
		if samples[i].HR == nil {
			continue
		}
		pct := float64(*samples[i].HR) / float64(hrMax)
		for z, zone := range trimpZones {
			if pct >= zone.lowerPct && pct < zone.upperPct {
				zoneSeconds[z]++
				break
			}
		}
	}

	// Edwards TRIMP = sum(zones: minutes_in_zone × zone_weight)
	var trimp float64
	for z := 0; z < 5; z++ {
		minutes := zoneSeconds[z] / 60.0
		trimp += minutes * trimpZones[z].weight
	}

	// Normalize to 0–100.
	rating := (trimp / refMaxTRIMP) * 100.0
	if rating > 100.0 {
		rating = 100.0
	}
	return &rating
}
