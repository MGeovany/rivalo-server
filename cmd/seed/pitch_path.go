package main

import (
	"math"
	"math/rand"
)

// Demo pitch anchor (Tegucigalpa area) — lat/lon span maps 0…1 pitch coords to GPS.
const (
	demoBaseLat  = 14.0723
	demoBaseLon  = -87.1921
	demoSpanLat  = 0.00072 // ~80 m width
	demoSpanLon  = 0.00011 // ~105 m length
)

// demoPathPoints returns a realistic, position-anchored trajectory as normalized
// (x,y) coordinates (0…1, attack → +X). Instead of a smooth loop (which renders
// as a ring), it is an Ornstein-Uhlenbeck random walk: the player drifts around a
// home zone with frequent direction changes, producing a natural cloud heatmap
// and a jagged route. Deterministic per sessionIndex.
func demoPathPoints(durationS, sessionIndex int, xHome, yHome float64) [][2]float64 {
	r := rand.New(rand.NewSource(int64(sessionIndex)*9973 + 7))
	const theta = 0.08 // pull back toward home
	const sigma = 0.06 // step volatility

	x, y := xHome, yHome
	points := make([][2]float64, 0, durationS/5+1)
	for offset := 0; offset <= durationS; offset += 5 {
		// Occasional attacking/defensive surge: shift the target forward briefly.
		targetX := xHome
		if r.Float64() < 0.08 {
			targetX = clamp01(xHome + (r.Float64()-0.3)*0.5)
		}
		x += theta*(targetX-x) + sigma*r.NormFloat64()
		y += theta*(yHome-y) + sigma*r.NormFloat64()
		points = append(points, [2]float64{clamp01(x), clamp01(y)})
	}
	return points
}

func pitchToGPS(x, y float64) (lat, lon float64) {
	return demoBaseLat + y*demoSpanLat, demoBaseLon + x*demoSpanLon
}

func demoSpeedKmh(tOffsetS, durationS int) float64 {
	base := 7.5 + 4.0*math.Sin(float64(tOffsetS)*0.004)
	// Sprint bursts every ~90s for 12s
	if (tOffsetS/90)%2 == 0 && tOffsetS%90 < 12 {
		return 21.0 + 4.0*math.Sin(float64(tOffsetS)*0.3)
	}
	if (tOffsetS/150)%3 == 1 && tOffsetS%150 < 8 {
		return 23.5
	}
	_ = durationS
	return base
}

func demoHR(tOffsetS, hrAvg int) int {
	return hrAvg - 18 + (tOffsetS/60)*2 + (tOffsetS%45)/3
}

func clamp01(v float64) float64 {
	if v < 0.08 {
		return 0.08
	}
	if v > 0.92 {
		return 0.92
	}
	return v
}
