package main

import (
	"log"

	"cyberfarm/internal/adapter/ebitengame"
	"cyberfarm/internal/usecase"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	svc := usecase.NewService()
	g := ebitengame.NewGame(svc)

	ebiten.SetWindowTitle("Киберпанк Ферма — Go/Ebitengine")
	w, h := g.Layout(0, 0)
	ebiten.SetWindowSize(int(float64(w)*1.05), int(float64(h)*1.05))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
