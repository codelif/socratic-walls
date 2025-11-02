package socraticwalls

import (
	"image/color"
	"math"
)

func hsvToNRGBA(h, s, v float64) color.NRGBA {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return color.NRGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func makeHSVPalette(size int) []color.NRGBA {
	p := make([]color.NRGBA, size)
	for i := range size {
		t := float64(i) / float64(size)
		h := t * 360.0
		p[i] = hsvToNRGBA(h, 1.0, 1.0)
	}
	return p
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + t*(float64(b)-float64(a)))
}

func lerpColor(c1, c2 color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: lerp(c1.R, c2.R, t),
		G: lerp(c1.G, c2.G, t),
		B: lerp(c1.B, c2.B, t),
		A: 255,
	}
}

// Stop is one gradient stop.
type Stop struct {
	T float64     `json:"t"`
	C color.NRGBA `json:"c"`
}

func makeGradient(stops []Stop) func(float64) color.NRGBA {
	if len(stops) == 0 {
		return func(float64) color.NRGBA {
			return color.NRGBA{0, 0, 0, 255}
		}
	}
	if len(stops) == 1 {
		base := stops[0].C
		return func(float64) color.NRGBA { return base }
	}

	return func(t float64) color.NRGBA {
		if t <= stops[0].T {
			return stops[0].C
		}
		lastIdx := len(stops) - 1
		if t >= stops[lastIdx].T {
			return stops[lastIdx].C
		}
		for i := range lastIdx {
			a := stops[i]
			b := stops[i+1]
			if t >= a.T && t <= b.T {
				span := b.T - a.T
				u := 0.0
				if span > 0 {
					u = (t - a.T) / span
				}
				return lerpColor(a.C, b.C, u)
			}
		}
		return stops[lastIdx].C
	}
}

type Sample struct {
	Iter   int
	Smooth float64
	Angle  float64
}

type ColorMode int

const (
	ColorSmoothHSV ColorMode = iota
	ColorLongGradient
	ColorPeriodic
	ColorAngle
	ColorHistogram
)

func buildColorizer(
	mode ColorMode,
	samples []Sample,
	maxIter int,
	hsvPalette []color.NRGBA,
	longGrad func(float64) color.NRGBA,
) func(int) color.NRGBA {

	black := color.NRGBA{0, 0, 0, 255}

	switch mode {

	case ColorSmoothHSV:
		return func(i int) color.NRGBA {
			s := samples[i]
			if s.Iter >= maxIter || s.Smooth < 0 {
				return black
			}
			base := math.Floor(s.Smooth)
			idx := int(base)
			frac := s.Smooth - base
			c1 := hsvPalette[idx%len(hsvPalette)]
			c2 := hsvPalette[(idx+1)%len(hsvPalette)]
			return lerpColor(c1, c2, frac)
		}

	case ColorLongGradient:
		return func(i int) color.NRGBA {
			s := samples[i]
			if s.Iter >= maxIter || s.Smooth < 0 {
				return black
			}
			t := math.Mod(s.Smooth*0.02, 1.0)
			return longGrad(t)
		}

	case ColorPeriodic:
		const period = 40.0
		return func(i int) color.NRGBA {
			s := samples[i]
			if s.Iter >= maxIter || s.Smooth < 0 {
				return black
			}
			t := math.Mod(s.Smooth, period) / period
			return longGrad(t)
		}

	case ColorAngle:
		return func(i int) color.NRGBA {
			s := samples[i]
			if s.Iter >= maxIter || s.Smooth < 0 {
				return black
			}
			escapeT := math.Mod(s.Smooth*0.03, 1.0)
			t := 0.6*s.Angle + 0.4*escapeT
			return longGrad(t)
		}

	case ColorHistogram:
		hist := make([]int, maxIter+1)
		totalEscaped := 0
		for _, s := range samples {
			if s.Iter < maxIter && s.Smooth >= 0 {
				hist[s.Iter]++
				totalEscaped++
			}
		}

		cdf := make([]float64, maxIter+1)
		run := 0
		for i := 0; i <= maxIter; i++ {
			run += hist[i]
			if totalEscaped > 0 {
				cdf[i] = float64(run) / float64(totalEscaped)
			} else {
				cdf[i] = 0
			}
		}

		return func(i int) color.NRGBA {
			s := samples[i]
			if s.Iter >= maxIter || s.Smooth < 0 {
				return black
			}

			base := math.Floor(s.Smooth)
			frac := s.Smooth - base
			k := max(int(base), 0)
			if k >= maxIter {
				k = maxIter - 1
			}

			t := cdf[k]
			if k+1 <= maxIter {
				t = cdf[k] + frac*(cdf[k+1]-cdf[k])
			}
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			return longGrad(t)
		}
	}

	return func(int) color.NRGBA { return black }
}
