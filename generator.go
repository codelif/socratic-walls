package socraticwalls

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"

	"socraticwalls/scorer"
)

func Map(value, oLow, oHi, nLow, nHi float64) float64 {
	return (value-oLow)*(nHi-nLow)/(oHi-oLow) + nLow
}

type GradientDef struct {
	Name  string `json:"name"`
	Stops []Stop `json:"stops"`
}

var DefaultGradients = []GradientDef{
	{
		Name: "Deep Ocean",
		Stops: []Stop{
			{0.00, color.NRGBA{0, 7, 100, 255}},
			{0.25, color.NRGBA{32, 107, 203, 255}},
			{0.50, color.NRGBA{237, 255, 255, 255}},
			{0.75, color.NRGBA{255, 170, 0, 255}},
			{1.00, color.NRGBA{0, 2, 0, 255}},
		},
	},
	{
		Name: "Inferno Ember",
		Stops: []Stop{
			{0.00, color.NRGBA{5, 0, 10, 255}},
			{0.20, color.NRGBA{120, 12, 40, 255}},
			{0.45, color.NRGBA{240, 60, 10, 255}},
			{0.70, color.NRGBA{255, 200, 50, 255}},
			{1.00, color.NRGBA{20, 2, 0, 255}},
		},
	},
	{
		Name: "Magenta Storm",
		Stops: []Stop{
			{0.00, color.NRGBA{15, 0, 40, 255}},
			{0.30, color.NRGBA{130, 0, 155, 255}},
			{0.60, color.NRGBA{255, 100, 180, 255}},
			{1.00, color.NRGBA{255, 230, 150, 255}},
		},
	},
	{
		Name: "Frostfire",
		Stops: []Stop{
			{0.00, color.NRGBA{0, 30, 50, 255}},
			{0.35, color.NRGBA{60, 190, 210, 255}},
			{0.65, color.NRGBA{255, 255, 255, 255}},
			{0.90, color.NRGBA{255, 140, 40, 255}},
			{1.00, color.NRGBA{20, 5, 0, 255}},
		},
	},
	{
		Name: "Verdant",
		Stops: []Stop{
			{0.00, color.NRGBA{0, 0, 0, 255}},
			{0.35, color.NRGBA{0, 90, 40, 255}},
			{0.65, color.NRGBA{160, 220, 40, 255}},
			{1.00, color.NRGBA{250, 255, 220, 255}},
		},
	},
}

func GradientByIndex(i int) func(float64) color.NRGBA {
	if i < 0 || i >= len(DefaultGradients) {
		return nil
	}
	return makeGradient(DefaultGradients[i].Stops)
}

func MakeGradientFromDef(def GradientDef) func(float64) color.NRGBA {
	return makeGradient(def.Stops)
}

func Generate(
	width, height int,
	v scorer.View,
	colorMode ColorMode,
	longGrad func(float64) color.NRGBA,
) image.Image {

	xLo, xHi, yLo, yHi := v.Bounds(width, height)

	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	hsvPalette := makeHSVPalette(1024)

	const maxIter = 1000
	const escapeR2 = 4.0

	samples := make([]Sample, width*height)

	nw := runtime.GOMAXPROCS(0)

	var wg sync.WaitGroup
	wg.Add(nw)
	for w := range nw {
		go func(worker int) {
			defer wg.Done()
			for py := worker; py < height; py += nw {
				y0 := Map(float64(py), 0, float64(height), yLo, yHi)
				rowOff := py * width
				for px := range width {
					x0 := Map(float64(px), 0, float64(width), xLo, xHi)

					x, y := 0.0, 0.0
					n := 0
					for x*x+y*y <= escapeR2 && n < maxIter {
						xx := x*x - y*y + x0
						y = 2*x*y + y0
						x = xx
						n++
					}

					s := Sample{Iter: n, Smooth: -1, Angle: 0}

					if n < maxIter {
						mod2 := x*x + y*y
						logZn := math.Log(mod2) / 2
						nu := math.Log(logZn/math.Log(2)) / math.Log(2)
						smooth := float64(n) + 1 - nu

						ang := math.Atan2(y, x)
						ang = (ang + math.Pi) / (2 * math.Pi)

						s.Smooth = smooth
						s.Angle = ang
					}

					samples[rowOff+px] = s
				}
			}
		}(w)
	}
	wg.Wait()

	if longGrad == nil {
		longGrad = makeGradient(DefaultGradients[0].Stops)
	}

	colorize := buildColorizer(colorMode, samples, maxIter, hsvPalette, longGrad)

	stride := img.Stride
	pix := img.Pix

	wg.Add(nw)
	for w := range nw {
		go func(worker int) {
			defer wg.Done()
			for py := worker; py < height; py += nw {
				rowPixOff := py * stride
				rowSampleOff := py * width
				for px := range width {
					idx := rowSampleOff + px
					c := colorize(idx)

					off := rowPixOff + px*4
					pix[off+0] = c.R
					pix[off+1] = c.G
					pix[off+2] = c.B
					pix[off+3] = c.A
				}
			}
		}(w)
	}
	wg.Wait()

	return img
}
