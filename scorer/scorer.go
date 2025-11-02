package scorer

import (
	"math"
	"math/rand"
)

type View struct {
	CX, CY float64
	Scale  float64
}

func (v View) Bounds(imgW, imgH int) (xMin, xMax, yMin, yMax float64) {
	aspect := float64(imgH) / float64(imgW)
	halfW := v.Scale / 2
	halfH := (v.Scale * aspect) / 2

	xMin = v.CX - halfW
	xMax = v.CX + halfW
	yMin = v.CY - halfH
	yMax = v.CY + halfH
	return xMin, xMax, yMin, yMax
}

func GenerateViews(n int) []ViewScore {
	var best []ViewScore

	for range 2000 {
		v := randomView()
		s := scoreView(v, 96, 54, 200)
		best = insertBest(best, ViewScore{V: v, Score: s}, n)
	}

	return best
}

type ViewScore struct {
	V     View
	Score float64
}

func insertBest(list []ViewScore, vs ViewScore, max int) []ViewScore {
	list = append(list, vs)
	for i := len(list) - 1; i > 0; i-- {
		if list[i].Score > list[i-1].Score {
			list[i], list[i-1] = list[i-1], list[i]
		} else {
			break
		}
	}
	if len(list) > max {
		return list[:max]
	}
	return list
}

func randomView() View {
	cx := rand.Float64()*3.5 - 2.5
	cy := rand.Float64()*3.0 - 1.5

	r := rand.Float64()
	r = r * r
	minScale := 0.0000005
	maxScale := 3.5
	scale := minScale + r*(maxScale-minScale)

	return View{CX: cx, CY: cy, Scale: scale}
}

func scoreView(v View, w, h, maxIter int) float64 {
	iters := make([]int, w*h)

	hScale := v.Scale * float64(h) / float64(w)
	x0 := v.CX - v.Scale/2
	y0 := v.CY - hScale/2

	var insideCount int
	maxR2 := 4.0

	for py := range h {
		cy := y0 + float64(py)/float64(h)*hScale
		for px := range w {
			cx := x0 + float64(px)/float64(w)*v.Scale

			x, y := 0.0, 0.0
			n := 0
			for x*x+y*y <= maxR2 && n < maxIter {
				xx := x*x - y*y + cx
				y = 2*x*y + cy
				x = xx
				n++
			}
			if n == maxIter {
				insideCount++
			}
			iters[py*w+px] = n
		}
	}

	total := float64(w * h)
	insideRatio := float64(insideCount) / total
	escapeScore := 1.0 - math.Abs(insideRatio-0.4)/0.4
	if escapeScore < 0 {
		escapeScore = 0
	}

	bins := 32
	hist := make([]int, bins)
	for _, n := range iters {
		b := n * bins / maxIter
		if b >= bins {
			b = bins - 1
		}
		hist[b]++
	}
	entropy := 0.0
	for _, c := range hist {
		if c == 0 {
			continue
		}
		p := float64(c) / total
		entropy -= p * math.Log(p)
	}
	entropyScore := entropy / math.Log(float64(bins))

	edgeCount := 0
	for py := range h {
		for px := range w {
			idx := py*w + px
			v0 := iters[idx]
			if px+1 < w {
				if absInt(v0-iters[idx+1]) > 2 {
					edgeCount++
				}
			}
			if py+1 < h {
				if absInt(v0-iters[idx+w]) > 2 {
					edgeCount++
				}
			}
		}
	}
	edgeScore := float64(edgeCount) / float64(2*w*h)

	return 0.5*entropyScore + 0.3*edgeScore + 0.2*escapeScore
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
