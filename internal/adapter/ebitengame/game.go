package ebitengame

import (
    "fmt"
    "image"
    "image/color"
    "time"

    "cyberfarm/internal/domain"
    "cyberfarm/internal/usecase"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "github.com/hajimehoshi/ebiten/v2/text"
)

type Game struct {
	svc *usecase.Service

	tile   int
	margin int

	faces   Faces
	buttons []Button
}

func NewGame(svc *usecase.Service) *Game {
	return &Game{svc: svc, tile: 32, margin: 8, faces: loadFaces()}
}

func (g *Game) Update() error {
    // Build buttons for the current frame
    sw, sh := g.Layout(0, 0)
    g.buildButtons(sw, sh)

    // Story event with mouse choices
    if st := g.svc.State(); st.PausedForEvent && st.CurrentEvent != nil {
        if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
            x, y := ebiten.CursorPosition()
            for _, b := range g.buttons {
                if b.Enabled && pointInRect(x, y, b.Rect) {
                    if b.OnClick != nil { b.OnClick() }
                    break
                }
            }
        }
        return nil
    }

    // Mouse interactions
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        x, y := ebiten.CursorPosition()
        if idx, ok := g.plotAtScreen(x, y); ok {
            g.svc.SelectPlot(idx)
        } else {
            for _, b := range g.buttons {
                if b.Enabled && pointInRect(x, y, b.Rect) {
                    if b.OnClick != nil { b.OnClick() }
                    break
                }
            }
        }
    }

    // Tick growth each second
    g.svc.MaybeTick(time.Now())
    return nil
}

type Button struct {
    Rect    image.Rectangle
    Label   string
    Enabled bool
    OnClick func()
}

func (g *Game) Draw(screen *ebiten.Image) {
	st := g.svc.State()
	// background
	screen.Fill(color.RGBA{0x0f, 0x0f, 0x15, 0xff})

	// grid
	ox, oy := g.margin, g.margin
	for i := 0; i < domain.TotalPlots; i++ {
		x := i % domain.FarmWidth
		y := i / domain.FarmWidth
		px := ox + x*(g.tile+2)
		py := oy + y*(g.tile+2)

		c := color.RGBA{0x16, 0x16, 0x22, 0xff}
		if i >= st.AvailablePlots || !st.Plots[i].Bought {
			c = color.RGBA{0x33, 0x33, 0x44, 0xff}
		}
		ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), float64(g.tile), c)

		// selection
		if st.SelectedPlot == i {
			ebitenutil.DrawRect(screen, float64(px-2), float64(py-2), float64(g.tile+4), float64(g.tile+4), color.RGBA{0xf3, 0xff, 0x00, 0x40})
		}

		// plant
		p := st.Plots[i]
		if p.HasPlant {
			pc := plantColor(p.Plant)
			if p.Harvestable {
				pc = color.RGBA{0xff, 0xd7, 0x00, 0xff}
			}
			if p.Watered && !p.Harvestable {
				// light blue overlay
				ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), float64(g.tile), color.RGBA{0x00, 0xe0, 0xff, 0x30})
			}
			// draw plant center rectangle sized by stage
			size := int(float64(g.tile) * (0.2 + 0.15*float64(p.GrowthStage)))
			if size > g.tile {
				size = g.tile
			}
			sx := px + (g.tile-size)/2
			sy := py + (g.tile-size)/2
			ebitenutil.DrawRect(screen, float64(sx), float64(sy), float64(size), float64(size), pc)
		}
	}

    // HUD
    hx := ox + domain.FarmWidth*(g.tile+2) + g.margin
    hy := oy
    white := color.RGBA{0xff, 0xff, 0xff, 0xff}
    titleCol := color.RGBA{0xff, 0x2d, 0x95, 0xff}
    text.Draw(screen, "Киберпанк Ферма", g.faces.Large, hx, hy+18, titleCol)
    text.Draw(screen, fmt.Sprintf("Дублоны: %d", st.Coins), g.faces.Normal, hx, hy+40, white)
    text.Draw(screen, fmt.Sprintf("Счёт: %.1f", st.TotalScore), g.faces.Normal, hx, hy+58, white)

    // Draw buttons
    for _, b := range g.buttons {
        g.drawButton(screen, b)
    }

    // Seed rows labels next to buttons
    y := g.margin + 70
    btnH, pad := 26, 8
    // skip 5 action buttons + spacing
    y += 5*(btnH+pad)
    y += 6
    // skip 3 upgrade buttons + spacing
    y += 3*(btnH+pad)
    y += pad
    text.Draw(screen, "Семена:", g.faces.Normal, hx, y-8, white)
    for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
        lock := ""
        if !st.UnlockedSeeds[pt] {
            lock = fmt.Sprintf(" (открыть: %d)", domain.SeedUnlockCosts[pt])
        } else {
            lock = fmt.Sprintf(" (цена: %d)", domain.SeedPrices[pt])
        }
        line := fmt.Sprintf("%s: %d%s", domain.PlantNames[pt], st.Seeds[pt], lock)
        text.Draw(screen, line, g.faces.Small, hx+2, y-10, white)
        y += btnH + 6
    }

	// Logs (bottom-left)
	lx := ox
	ly := oy + domain.FarmHeight*(g.tile+2) + 10
    text.Draw(screen, "Журнал:", g.faces.Normal, lx, ly, white)
	ly += 16
	max := 8
	for i := 0; i < len(st.Log) && i < max; i++ {
        text.Draw(screen, st.Log[i], g.faces.Small, lx, ly, white)
		ly += 14
	}

	// Story event overlay
	if st.PausedForEvent && st.CurrentEvent != nil {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		ebitenutil.DrawRect(screen, 0, 0, float64(w), float64(h), color.RGBA{0, 0, 0, 160})
        cx := w/2 - 220
        cy := h/2 - 80
        text.Draw(screen, "Событие:", g.faces.Large, cx, cy, white)
        text.Draw(screen, st.CurrentEvent.Message, g.faces.Normal, cx, cy+22, white)
	}

	// Win overlay
	if st.Won {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		ebitenutil.DrawRect(screen, 0, 0, float64(w), float64(h), color.RGBA{0, 0, 0, 160})
        msg := fmt.Sprintf("Победа! Счёт: %.1f", st.TotalScore)
        text.Draw(screen, msg, g.faces.Large, w/2-120, h/2, color.RGBA{0xf3, 0xff, 0x00, 0xff})
    }
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// Compute needed size: grid + HUD
	gridW := domain.FarmWidth*(g.tile+2) + g.margin*2
	gridH := domain.FarmHeight*(g.tile+2) + g.margin*2 + 160 // space for logs
	hudW := 360
	return gridW + hudW, gridH
}

func plantColor(pt domain.PlantType) color.RGBA {
	switch pt {
	case domain.PlantTomato:
		return color.RGBA{0xff, 0x00, 0x00, 0xff}
	case domain.PlantCucumber:
		return color.RGBA{0x00, 0xff, 0x94, 0xff}
	case domain.PlantPotato:
		return color.RGBA{0x96, 0x6f, 0x33, 0xff}
	case domain.PlantCarrot:
		return color.RGBA{0xff, 0x80, 0x00, 0xff}
	case domain.PlantTurnip:
		return color.RGBA{0xaa, 0x88, 0xff, 0xff}
	case domain.PlantCabbage:
		return color.RGBA{0x44, 0xff, 0x44, 0xff}
	case domain.PlantBeet:
		return color.RGBA{0xcc, 0x00, 0x44, 0xff}
	case domain.PlantSunflower:
		return color.RGBA{0xff, 0xcc, 0x00, 0xff}
	default:
		return color.RGBA{0xff, 0xff, 0xff, 0xff}
	}
}

func yesNo(b bool) string {
	if b {
		return "есть"
	}
	return "нет"
}

// ---- UI helpers (mouse buttons, layout, text) ----

func (g *Game) drawButton(screen *ebiten.Image, b Button) {
    col := color.RGBA{0x1a, 0x1a, 0x26, 0xff}
    border := color.RGBA{0x00, 0xe0, 0xff, 0xff}
    txt := color.RGBA{0xff, 0xff, 0xff, 0xff}
    if !b.Enabled {
        col = color.RGBA{0x33, 0x33, 0x44, 0xff}
        border = color.RGBA{0x55, 0x55, 0x66, 0xff}
        txt = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
    }
    ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), float64(b.Rect.Dx()), float64(b.Rect.Dy()), col)
    ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), float64(b.Rect.Dx()), 1, border)
    ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Max.Y-1), float64(b.Rect.Dx()), 1, border)
    ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), 1, float64(b.Rect.Dy()), border)
    ebitenutil.DrawRect(screen, float64(b.Rect.Max.X-1), float64(b.Rect.Min.Y), 1, float64(b.Rect.Dy()), border)
    // label centered
    bounds := text.BoundString(g.faces.Normal, b.Label)
    tw := bounds.Dx()
    th := bounds.Dy()
    tx := b.Rect.Min.X + (b.Rect.Dx()-tw)/2
    ty := b.Rect.Min.Y + (b.Rect.Dy()+th)/2 - 2
    text.Draw(screen, b.Label, g.faces.Normal, tx, ty, txt)
}

func pointInRect(x, y int, r image.Rectangle) bool {
    return x >= r.Min.X && y >= r.Min.Y && x < r.Max.X && y < r.Max.Y
}

func (g *Game) plotAtScreen(x, y int) (int, bool) {
    ox, oy := g.margin, g.margin
    for i := 0; i < domain.TotalPlots; i++ {
        px := ox + (i%domain.FarmWidth)*(g.tile+2)
        py := oy + (i/domain.FarmWidth)*(g.tile+2)
        r := image.Rect(px, py, px+g.tile, py+g.tile)
        if pointInRect(x, y, r) {
            st := g.svc.State()
            if i < st.AvailablePlots && st.Plots[i].Bought {
                return i, true
            }
            return -1, false
        }
    }
    return -1, false
}

func (g *Game) buildButtons(sw, sh int) {
    st := g.svc.State()
    g.buttons = g.buttons[:0]

    if st.PausedForEvent && st.CurrentEvent != nil {
        cx := sw/2 - 200
        cy := sh/2 - 10
        g.buttons = append(g.buttons,
            Button{Rect: image.Rect(cx, cy, cx+180, cy+28), Label: st.CurrentEvent.Choices[0].Text, Enabled: true, OnClick: func(){ g.svc.ResolveEvent(0) }},
            Button{Rect: image.Rect(cx, cy+36, cx+180, cy+64), Label: st.CurrentEvent.Choices[1].Text, Enabled: true, OnClick: func(){ g.svc.ResolveEvent(1) }},
        )
        return
    }

    hx := g.margin + domain.FarmWidth*(g.tile+2) + g.margin
    y := g.margin + 70

    btnW, btnH, pad := 160, 26, 8
    add := func(label string, enabled bool, cb func()) {
        g.buttons = append(g.buttons, Button{Rect: image.Rect(hx, y, hx+btnW, y+btnH), Label: label, Enabled: enabled, OnClick: cb})
        y += btnH + pad
    }

    canPlant := st.HasSelectedSeed && st.Seeds[st.SelectedSeed] > 0 && ((st.CanPlantBulk && hasAnyEmptyBought(st)) || (!st.CanPlantBulk && st.SelectedPlot >= 0 && st.SelectedPlot < len(st.Plots) && !st.Plots[st.SelectedPlot].HasPlant && st.Plots[st.SelectedPlot].Bought))
    add("Посадить", canPlant, func(){ g.svc.Plant() })

    canWater := (st.CanWaterBulk && hasAnyWaterable(st)) || (!st.CanWaterBulk && st.SelectedPlot >= 0 && st.Plots[st.SelectedPlot].HasPlant && !st.Plots[st.SelectedPlot].Watered && !st.Plots[st.SelectedPlot].Harvestable)
    add("Полить", canWater, func(){ g.svc.Water() })

    canHarvest := (st.CanHarvestBulk && hasAnyHarvestable(st)) || (!st.CanHarvestBulk && st.SelectedPlot >= 0 && st.Plots[st.SelectedPlot].Harvestable)
    add("Собрать", canHarvest, func(){ g.svc.Harvest() })

    canSell := hasAnyProduce(st)
    add("Продать всё", canSell, func(){ g.svc.SellAll() })

    canExpand := st.AvailablePlots < domain.TotalPlots && st.Coins >= domain.ExpandPlotCost(st.AvailablePlots)
    add("Купить клетку", canExpand, func(){ g.svc.ExpandField() })

    // upgrades
    y += 6
    add(fmt.Sprintf("Засев x4 (50) [%s]", yesNo(st.CanPlantBulk)), !st.CanPlantBulk && st.Coins >= 50, func(){ g.svc.BuyUpgradePlantBulk() })
    add(fmt.Sprintf("Полив x4 (30) [%s]", yesNo(st.CanWaterBulk)), !st.CanWaterBulk && st.Coins >= 30, func(){ g.svc.BuyUpgradeWaterBulk() })
    add(fmt.Sprintf("Сбор x4 (40) [%s]", yesNo(st.CanHarvestBulk)), !st.CanHarvestBulk && st.Coins >= 40, func(){ g.svc.BuyUpgradeHarvestBulk() })

    // seeds rows
    y += pad
    for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
        selectEnabled := st.UnlockedSeeds[pt] && st.Seeds[pt] > 0
        g.buttons = append(g.buttons, Button{Rect: image.Rect(hx, y, hx+96, y+btnH), Label: "Выбрать", Enabled: selectEnabled, OnClick: func(p domain.PlantType) func(){ return func(){ g.svc.SelectSeed(p) } }(pt)})
        var label string
        var enabled bool
        if !st.UnlockedSeeds[pt] {
            label = fmt.Sprintf("Открыть (%d)", domain.SeedUnlockCosts[pt])
            enabled = st.Coins >= domain.SeedUnlockCosts[pt]
        } else {
            label = fmt.Sprintf("Купить (%d)", domain.SeedPrices[pt])
            enabled = st.Coins >= domain.SeedPrices[pt]
        }
        g.buttons = append(g.buttons, Button{Rect: image.Rect(hx+100, y, hx+100+120, y+btnH), Label: label, Enabled: enabled, OnClick: func(p domain.PlantType) func(){ return func(){ g.svc.BuySeedOrUnlock(p) } }(pt)})

        if st.HasSelectedSeed && st.SelectedSeed == pt {
            g.buttons = append(g.buttons, Button{Rect: image.Rect(hx+224, y, hx+224+80, y+btnH), Label: "[выбрано]", Enabled: false})
        }
        y += btnH + 6
    }
}

func hasAnyEmptyBought(st *domain.GameState) bool {
    for i := 0; i < st.AvailablePlots; i++ {
        if st.Plots[i].Bought && !st.Plots[i].HasPlant {
            return true
        }
    }
    return false
}

func hasAnyWaterable(st *domain.GameState) bool {
    for i := 0; i < st.AvailablePlots; i++ {
        p := st.Plots[i]
        if p.Bought && p.HasPlant && !p.Watered && !p.Harvestable {
            return true
        }
    }
    return false
}

func hasAnyHarvestable(st *domain.GameState) bool {
    for i := 0; i < st.AvailablePlots; i++ {
        p := st.Plots[i]
        if p.Bought && p.Harvestable {
            return true
        }
    }
    return false
}

func hasAnyProduce(st *domain.GameState) bool {
    for _, v := range st.Produce {
        if v > 0 { return true }
    }
    return false
}
