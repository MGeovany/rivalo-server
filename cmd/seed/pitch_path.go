package main

import "math"

// Demo pitch anchor (Tegucigalpa area) — lat/lon span maps 0…1 pitch coords to GPS.
const (
	demoBaseLat  = 14.0723
	demoBaseLon  = -87.1921
	demoSpanLat  = 0.00072 // ~80 m width
	demoSpanLon  = 0.00011 // ~105 m length
)

// demoPitchXY returns normalized pitch coordinates (0…1) with attack → +X (right).
// Layout matches the reference heatmap: hot zones bottom-right and center-right.
func demoPitchXY(tOffsetS, durationS, sessionIndex int, posBias float64) (x, y float64) {
	if durationS <= 0 {
		return clamp01(0.5 + posBias), 0.5
	}
	progress := float64(tOffsetS) / float64(durationS)
	t := float64(tOffsetS)

	// Slight variation per session so charts are not identical.
	shift := float64(sessionIndex) * 0.04

	var cx, cy float64
	switch {
	case progress < 0.12:
		cx, cy = 0.28+shift*0.3, 0.48 // defensive left
	case progress < 0.28:
		cx, cy = 0.42, 0.55+math.Sin(t*0.02)*0.06 // build-up
	case progress < 0.45:
		cx, cy = 0.58, 0.42 // midfield lane
	case progress < 0.62:
		cx, cy = 0.70, 0.38 // center-right band
	case progress < 0.78:
		cx, cy = 0.78, 0.58 // attacking third
	default:
		cx, cy = 0.82, 0.70 // bottom-right hot zone
	}

	// Linger in high-density zones (reference red clusters).
	if progress > 0.5 && progress < 0.85 {
		cx += math.Sin(t*0.035)*0.06 + 0.04
		cy += math.Cos(t*0.028)*0.07
	}
	if progress > 0.35 && progress < 0.55 {
		cx += 0.02
		cy += math.Sin(t*0.04)*0.05
	}

	jitterX := math.Sin(t*0.11)*0.03 + math.Cos(t*0.07)*0.02
	jitterY := math.Cos(t*0.09)*0.04 + math.Sin(t*0.13)*0.02

	x = clamp01(cx + jitterX + posBias)
	y = clamp01(cy + jitterY)
	return x, y
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
