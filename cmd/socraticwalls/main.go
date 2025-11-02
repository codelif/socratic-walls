package main

import (
	"image/png"
	"log"
	"os"
	"socraticwalls"
	"socraticwalls/scorer"
)

func main() {
	best := scorer.GenerateViews(1)
	v := best[0].V
	img := socraticwalls.Generate(2880, 1800, v, socraticwalls.ColorHistogram)
	f, err := os.Create("mandel.png")
	if err != nil {
		log.Fatalln("could not create file:", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatalln("could not encode png:", err)
	}
}
