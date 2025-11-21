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
    // window size is determined by Layout

    if err := ebiten.RunGame(g); err != nil {
        log.Fatal(err)
    }
}

