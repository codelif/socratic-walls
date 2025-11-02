package main

import (
	"encoding/json"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"socraticwalls"
	"socraticwalls/scorer"
)

type viewPayload struct {
	CX    float64 `json:"cx"`
	CY    float64 `json:"cy"`
	Scale float64 `json:"scale"`
}

type mandelRequest struct {
	Width   int                        `json:"width"`
	Height  int                        `json:"height"`
	CX      float64                    `json:"cx"`
	CY      float64                    `json:"cy"`
	Scale   float64                    `json:"scale"`
	Mode    int                        `json:"mode"`
	Palette *socraticwalls.GradientDef `json:"palette,omitempty"`
}

var palettesMu sync.RWMutex

func qf(r *http.Request, key string, def float64) float64 {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func qi(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func randomViewHandler(w http.ResponseWriter, r *http.Request) {
	best := scorer.GenerateViews(1)
	v := best[0].V

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(viewPayload{
		CX:    v.CX,
		CY:    v.CY,
		Scale: v.Scale,
	})
}

func palettesHandler(w http.ResponseWriter, r *http.Request) {
	palettesMu.RLock()
	defer palettesMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(socraticwalls.DefaultGradients)
}


// hslToNRGBA is local here; if you prefer, move to socraticwalls/colors.go
func hslToNRGBA(h, s, l float64) color.NRGBA {
	// h: 0..360, s,l: 0..1
	c := (1 - math.Abs(2*l-1)) * s
	hp := h / 60.0
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch {
	case 0 <= hp && hp < 1:
		r, g, b = c, x, 0
	case 1 <= hp && hp < 2:
		r, g, b = x, c, 0
	case 2 <= hp && hp < 3:
		r, g, b = 0, c, x
	case 3 <= hp && hp < 4:
		r, g, b = 0, x, c
	case 4 <= hp && hp < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return color.NRGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}

func wrapHue(h float64) float64 {
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}
	return h
}

func randomPaletteDef() socraticwalls.GradientDef {
	// always 5 stops
	const n = 5
	stops := make([]socraticwalls.Stop, n)

	baseHue := rand.Float64() * 360.0

	// pick a spread so colors don't bunch up
	// 45..85 degrees
	spread := 45.0 + rand.Float64()*40.0

	// we’ll make 3 related hues and 2 distant ones
	hues := []float64{
		baseHue,
		baseHue + spread,
		baseHue + 2*spread,
		baseHue + 180,             // complementary
		baseHue + 180 + spread/2,  // slight variation on the complement
	}

	// make sure all in 0..360
	for i := range hues {
		hues[i] = wrapHue(hues[i])
	}

	// sane sat/light so it doesn’t go grey/muddy
	// you can randomize in a tighter range if you like
	s := 0.55 + rand.Float64()*0.35 // 0.55 .. 0.90
	l := 0.40 + rand.Float64()*0.20 // 0.40 .. 0.60

	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1) // 0, .25, .5, .75, 1
		c := hslToNRGBA(hues[i], s, l)
		stops[i] = socraticwalls.Stop{T: t, C: c}
	}

	return socraticwalls.GradientDef{
		Name:  "Random " + strconv.Itoa(int(time.Now().UnixNano())%10000),
		Stops: stops,
	}
}

// func randomPaletteDef() socraticwalls.GradientDef {
// 	n := rand.Intn(3) + 3 // 3..5 stops
// 	stops := make([]socraticwalls.Stop, n)
// 	for i := range n {
// 		t := float64(i) / float64(n-1)
// 		rCol := uint8(80 + rand.Intn(170))
// 		gCol := uint8(60 + rand.Intn(170))
// 		bCol := uint8(60 + rand.Intn(170))
// 		stops[i] = socraticwalls.Stop{T: t, C: color.NRGBA{rCol, gCol, bCol, 255}}
// 	}
// 	sort.Slice(stops, func(i, j int) bool { return stops[i].T < stops[j].T })
// 	return socraticwalls.GradientDef{
// 		Name:  "Random " + strconv.Itoa(int(time.Now().UnixNano())%10000),
// 		Stops: stops,
// 	}
// }

func randomPaletteHandler(w http.ResponseWriter, r *http.Request) {
	p := randomPaletteDef()

	palettesMu.Lock()
	socraticwalls.DefaultGradients = append(socraticwalls.DefaultGradients, p)
	idx := len(socraticwalls.DefaultGradients) - 1
	palettesMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		socraticwalls.GradientDef
		Index int `json:"index"`
	}{
		GradientDef: p,
		Index:       idx,
	})
}

func mandelGETHandler(w http.ResponseWriter, r *http.Request) {
	width := qi(r, "width", 1080)
	height := qi(r, "height", 660)

	cx := qf(r, "cx", -0.5)
	cy := qf(r, "cy", 0.0)
	scale := qf(r, "scale", 3.5)

	modeInt := qi(r, "mode", int(socraticwalls.ColorHistogram))
	if modeInt < int(socraticwalls.ColorSmoothHSV) || modeInt > int(socraticwalls.ColorHistogram) {
		modeInt = int(socraticwalls.ColorHistogram)
	}
	colorMode := socraticwalls.ColorMode(modeInt)

	paletteIdx := qi(r, "palette", 0)

	palettesMu.RLock()
	var grad func(float64) color.NRGBA
	if g := socraticwalls.GradientByIndex(paletteIdx); g != nil {
		grad = g
	}
	palettesMu.RUnlock()

	v := scorer.View{CX: cx, CY: cy, Scale: scale}
	img := socraticwalls.Generate(width, height, v, colorMode, grad)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_ = png.Encode(w, img)
}

func mandelPOSTHandler(w http.ResponseWriter, r *http.Request) {
	var req mandelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	width := req.Width
	if width <= 0 {
		width = 1080
	}
	height := req.Height
	if height <= 0 {
		height = 660
	}

	if req.Scale == 0 {
		req.Scale = 3.5
	}

	mode := socraticwalls.ColorMode(req.Mode)
	if mode < socraticwalls.ColorSmoothHSV || mode > socraticwalls.ColorHistogram {
		mode = socraticwalls.ColorHistogram
	}

	var grad func(float64) color.NRGBA
	if req.Palette != nil {
		grad = socraticwalls.MakeGradientFromDef(*req.Palette)
	}

	v := scorer.View{CX: req.CX, CY: req.CY, Scale: req.Scale}
	img := socraticwalls.Generate(width, height, v, mode, grad)

	w.Header().Set("Content-Type", "image/png")
	_ = png.Encode(w, img)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/view/random", randomViewHandler)
	mux.HandleFunc("/api/palettes", palettesHandler)
	mux.HandleFunc("/api/palettes/random", randomPaletteHandler)
	mux.HandleFunc("/api/mandel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mandelPOSTHandler(w, r)
			return
		}
		mandelGETHandler(w, r)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
