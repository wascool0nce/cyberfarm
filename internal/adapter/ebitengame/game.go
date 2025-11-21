package ebitengame

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"cyberfarm/internal/domain"
	"cyberfarm/internal/usecase"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

type Game struct {
	svc *usecase.Service

	tile     int
	margin   int
	hudWidth int

	btnW   int
	btnH   int
	btnPad int

	faces   Faces
	buttons []Button

	hoveredButton int
	hoveredPlot   int
	animPhase     float64
	lastFrame     time.Time
}

func NewGame(svc *usecase.Service) *Game {
	return &Game{
		svc:           svc,
		tile:          42,
		margin:        16,
		hudWidth:      520,
		btnW:          240,
		btnH:          36,
		btnPad:        12,
		faces:         loadFaces(),
		hoveredButton: -1,
		hoveredPlot:   -1,
	}
}

func (g *Game) Update() error {
	// Build buttons for the current frame
	sw, sh := g.Layout(0, 0)
	g.buildButtons(sw, sh)

	now := time.Now()
	if g.lastFrame.IsZero() {
		g.lastFrame = now
	}
	dt := now.Sub(g.lastFrame).Seconds()
	g.lastFrame = now
	g.animPhase += dt

	g.hoveredButton = -1
	g.hoveredPlot = -1
	mx, my := ebiten.CursorPosition()
	if idx, ok := g.plotAtScreen(mx, my); ok {
		g.hoveredPlot = idx
	} else {
		for i, b := range g.buttons {
			if pointInRect(mx, my, b.Rect) && b.Enabled {
				g.hoveredButton = i
				break
			}
		}
	}

	// Story event with mouse choices
	if st := g.svc.State(); st.PausedForEvent && st.CurrentEvent != nil {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			for _, b := range g.buttons {
				if b.Enabled && pointInRect(x, y, b.Rect) {
					if b.OnClick != nil {
						b.OnClick()
					}
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
					if b.OnClick != nil {
						b.OnClick()
					}
					break
				}
			}
		}
	}

	// Tick growth each second
	g.svc.MaybeTick(now)
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
	screen.Fill(color.RGBA{0x06, 0x08, 0x0f, 0xff})
	glowLayer := uint8(20 + 15*math.Sin(g.animPhase*1.5))
	ebitenutil.DrawRect(screen, 0, 0, float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy()), color.RGBA{0x07, 0x10, 0x24, glowLayer})

	ox, oy := g.margin, g.margin
	for i := 0; i < domain.TotalPlots; i++ {
		x := i % domain.FarmWidth
		y := i / domain.FarmWidth
		px := ox + x*(g.tile+2)
		py := oy + y*(g.tile+2)

		vibrate := 0.08 + 0.08*math.Sin(g.animPhase*2+float64(x+y))
		c := color.RGBA{uint8(float64(0x16) + 60*vibrate), uint8(float64(0x1b) + 40*vibrate), 0x2b, 0xff}
		if i >= st.AvailablePlots || !st.Plots[i].Bought {
			c = color.RGBA{0x26, 0x26, 0x32, 0xff}
		}
		if g.hoveredPlot == i {
			c = color.RGBA{0x22, 0x33, 0x4a, 0xff}
		}
		ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), float64(g.tile), c)

		if st.SelectedPlot == i {
			glow := uint8(80 + 40*math.Sin(g.animPhase*5))
			ebitenutil.DrawRect(screen, float64(px-3), float64(py-3), float64(g.tile+6), float64(g.tile+6), color.RGBA{0x00, 0xff, 0xe1, glow})
		}
		if g.hoveredPlot == i {
			ebitenutil.DrawRect(screen, float64(px-2), float64(py-2), float64(g.tile+4), float64(g.tile+4), color.RGBA{0x3a, 0xff, 0xc8, 0x35})
		}

		p := st.Plots[i]
		if p.HasPlant {
			pc := plantColor(p.Plant)
			if p.Harvestable {
				pulse := 0.7 + 0.25*math.Sin(g.animPhase*4+float64(i))
				pc = color.RGBA{uint8(255 * pulse), uint8(210 + 30*pulse), 0x10, 0xff}
			}
			if p.Watered && !p.Harvestable {
				ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), float64(g.tile), color.RGBA{0x00, 0xb8, 0xff, 0x40})
			}
			size := int(float64(g.tile) * (0.25 + 0.16*float64(p.GrowthStage)))
			if size > g.tile {
				size = g.tile
			}
			wobble := int(1 + 2*math.Sin(g.animPhase*3+float64(i)))
			sx := px + (g.tile-size)/2
			sy := py + (g.tile-size)/2 - wobble
			ebitenutil.DrawRect(screen, float64(sx), float64(sy), float64(size), float64(size), pc)
		}
	}

	hx := ox + domain.FarmWidth*(g.tile+2) + g.margin
	hy := oy
	white := color.RGBA{0xe8, 0xff, 0xff, 0xff}
	accent := color.RGBA{0x82, 0xff, 0xf9, 0xff}
	drawNeonText(screen, "Киберпанк Ферма", g.faces.Large, hx, hy+28, accent)
	drawNeonText(screen, fmt.Sprintf("Дублоны: %d", st.Coins), g.faces.Normal, hx, hy+62, white)
	drawNeonText(screen, fmt.Sprintf("Счёт: %.1f", st.TotalScore), g.faces.Normal, hx, hy+86, white)

	for i, b := range g.buttons {
		g.drawButton(screen, b, i == g.hoveredButton)
	}

	y := g.seedsStartY()
	drawNeonText(screen, "Семена:", g.faces.Normal, hx, y-10, white)
	for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
		lock := ""
		if !st.UnlockedSeeds[pt] {
			lock = fmt.Sprintf(" (открыть: %d)", domain.SeedUnlockCosts[pt])
		} else {
			lock = fmt.Sprintf(" (цена: %d)", domain.SeedPrices[pt])
		}
		line := fmt.Sprintf("%s: %d%s", domain.PlantNames[pt], st.Seeds[pt], lock)
		text.Draw(screen, line, g.faces.Small, hx+2, y-6, white)
		y += g.btnH + g.btnPad
	}

	lx := ox
	ly := oy + domain.FarmHeight*(g.tile+2) + 18
	drawNeonText(screen, "Журнал:", g.faces.Normal, lx, ly, white)
	ly += 18
	max := 10
	for i := 0; i < len(st.Log) && i < max; i++ {
		text.Draw(screen, st.Log[i], g.faces.Small, lx, ly, white)
		ly += 16
	}

	if st.PausedForEvent && st.CurrentEvent != nil {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		ebitenutil.DrawRect(screen, 0, 0, float64(w), float64(h), color.RGBA{0, 0, 0, 180})
		eventW := g.btnW
		cx := w/2 - eventW/2
		cy := h/2 - g.btnH - g.btnPad
		drawNeonText(screen, "Событие:", g.faces.Large, cx, cy-38, white)
		text.Draw(screen, st.CurrentEvent.Message, g.faces.Normal, cx, cy-10, white)
	}

	if st.Won {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		ebitenutil.DrawRect(screen, 0, 0, float64(w), float64(h), color.RGBA{0, 0, 0, 170})
		msg := fmt.Sprintf("Победа! Счёт: %.1f", st.TotalScore)
		drawNeonText(screen, msg, g.faces.Large, w/2-150, h/2, color.RGBA{0xf3, 0xff, 0x00, 0xff})
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	gridW := domain.FarmWidth*(g.tile+2) + g.margin*2
	gridH := domain.FarmHeight*(g.tile+2) + g.margin*2 + 200
	return gridW + g.hudWidth, gridH
}

func drawNeonText(screen *ebiten.Image, str string, face font.Face, x, y int, main color.Color) {
	shadow := color.RGBA{0x00, 0xff, 0xff, 0x60}
	accent := color.RGBA{0xff, 0x2d, 0x95, 0x70}
	text.Draw(screen, str, face, x-1, y-1, shadow)
	text.Draw(screen, str, face, x+1, y+1, accent)
	text.Draw(screen, str, face, x, y, main)
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

func (g *Game) drawButton(screen *ebiten.Image, b Button, hovered bool) {
	col := color.RGBA{0x14, 0x16, 0x24, 0xff}
	border := color.RGBA{0x00, 0xc8, 0xff, 0xff}
	txt := color.RGBA{0xe8, 0xff, 0xff, 0xff}
	if !b.Enabled {
		col = color.RGBA{0x33, 0x33, 0x44, 0xff}
		border = color.RGBA{0x55, 0x55, 0x66, 0xff}
		txt = color.RGBA{0xaa, 0xaa, 0xaa, 0xff}
	}
	if hovered && b.Enabled {
		boost := uint8(15 + 10*math.Sin(g.animPhase*5))
		add := func(v uint8) uint8 {
			s := int(v) + int(boost)
			if s > 255 {
				s = 255
			}
			return uint8(s)
		}
		col = color.RGBA{add(col.R), add(col.G), add(col.B), col.A}
	}
	ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), float64(b.Rect.Dx()), float64(b.Rect.Dy()), col)
	ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), float64(b.Rect.Dx()), 2, border)
	ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Max.Y-2), float64(b.Rect.Dx()), 2, border)
	ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), 2, float64(b.Rect.Dy()), border)
	ebitenutil.DrawRect(screen, float64(b.Rect.Max.X-2), float64(b.Rect.Min.Y), 2, float64(b.Rect.Dy()), border)

	if hovered && b.Enabled {
		glow := uint8(40 + 40*math.Sin(g.animPhase*6))
		ebitenutil.DrawRect(screen, float64(b.Rect.Min.X-2), float64(b.Rect.Min.Y-2), float64(b.Rect.Dx()+4), float64(b.Rect.Dy()+4), color.RGBA{border.R, border.G, border.B, glow})
	}
	face := g.faces.Normal
	bounds := text.BoundString(face, b.Label)
	if bounds.Dx() > b.Rect.Dx()-12 {
		face = g.faces.Small
		bounds = text.BoundString(face, b.Label)
	}
	tw := bounds.Dx()
	th := bounds.Dy()
	tx := b.Rect.Min.X + (b.Rect.Dx()-tw)/2
	ty := b.Rect.Min.Y + (b.Rect.Dy()+th)/2 - 2
	text.Draw(screen, b.Label, face, tx, ty, txt)
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
		eventW := g.btnW
		eventH := g.btnH
		cx := sw/2 - eventW/2
		cy := sh/2 - eventH
		g.buttons = append(g.buttons,
			Button{Rect: image.Rect(cx, cy, cx+eventW, cy+eventH), Label: st.CurrentEvent.Choices[0].Text, Enabled: true, OnClick: func() { g.svc.ResolveEvent(0) }},
			Button{Rect: image.Rect(cx, cy+eventH+g.btnPad, cx+eventW, cy+eventH*2+g.btnPad), Label: st.CurrentEvent.Choices[1].Text, Enabled: true, OnClick: func() { g.svc.ResolveEvent(1) }},
		)
		return
	}

	hx := g.margin + domain.FarmWidth*(g.tile+2) + g.margin
	y := g.actionStartY()

	add := func(label string, enabled bool, cb func()) {
		g.buttons = append(g.buttons, Button{Rect: image.Rect(hx, y, hx+g.btnW, y+g.btnH), Label: label, Enabled: enabled, OnClick: cb})
		y += g.btnH + g.btnPad
	}

	canPlant := st.HasSelectedSeed && st.Seeds[st.SelectedSeed] > 0 && ((st.CanPlantBulk && hasAnyEmptyBought(st)) || (!st.CanPlantBulk && st.SelectedPlot >= 0 && st.SelectedPlot < len(st.Plots) && !st.Plots[st.SelectedPlot].HasPlant && st.Plots[st.SelectedPlot].Bought))
	add("Посадить", canPlant, func() { g.svc.Plant() })

	canWater := (st.CanWaterBulk && hasAnyWaterable(st)) || (!st.CanWaterBulk && st.SelectedPlot >= 0 && st.Plots[st.SelectedPlot].HasPlant && !st.Plots[st.SelectedPlot].Watered && !st.Plots[st.SelectedPlot].Harvestable)
	add("Полить", canWater, func() { g.svc.Water() })

	canHarvest := (st.CanHarvestBulk && hasAnyHarvestable(st)) || (!st.CanHarvestBulk && st.SelectedPlot >= 0 && st.Plots[st.SelectedPlot].Harvestable)
	add("Собрать", canHarvest, func() { g.svc.Harvest() })

	canSell := hasAnyProduce(st)
	add("Продать всё", canSell, func() { g.svc.SellAll() })

	canExpand := st.AvailablePlots < domain.TotalPlots && st.Coins >= domain.ExpandPlotCost(st.AvailablePlots)
	add("Купить клетку", canExpand, func() { g.svc.ExpandField() })

	// upgrades
	y += g.btnPad
	add(fmt.Sprintf("Засев x4 (50) [%s]", yesNo(st.CanPlantBulk)), !st.CanPlantBulk && st.Coins >= 50, func() { g.svc.BuyUpgradePlantBulk() })
	add(fmt.Sprintf("Полив x4 (30) [%s]", yesNo(st.CanWaterBulk)), !st.CanWaterBulk && st.Coins >= 30, func() { g.svc.BuyUpgradeWaterBulk() })
	add(fmt.Sprintf("Сбор x4 (40) [%s]", yesNo(st.CanHarvestBulk)), !st.CanHarvestBulk && st.Coins >= 40, func() { g.svc.BuyUpgradeHarvestBulk() })

	// seeds rows
	y += g.btnPad
	for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
		selectEnabled := st.UnlockedSeeds[pt] && st.Seeds[pt] > 0
		selectW := 132
		buyW := 172
		statusW := 112
		g.buttons = append(g.buttons, Button{Rect: image.Rect(hx, y, hx+selectW, y+g.btnH), Label: "Выбрать", Enabled: selectEnabled, OnClick: func(p domain.PlantType) func() { return func() { g.svc.SelectSeed(p) } }(pt)})
		var label string
		var enabled bool
		if !st.UnlockedSeeds[pt] {
			label = fmt.Sprintf("Открыть (%d)", domain.SeedUnlockCosts[pt])
			enabled = st.Coins >= domain.SeedUnlockCosts[pt]
		} else {
			label = fmt.Sprintf("Купить (%d)", domain.SeedPrices[pt])
			enabled = st.Coins >= domain.SeedPrices[pt]
		}
		g.buttons = append(g.buttons, Button{Rect: image.Rect(hx+selectW+g.btnPad, y, hx+selectW+g.btnPad+buyW, y+g.btnH), Label: label, Enabled: enabled, OnClick: func(p domain.PlantType) func() { return func() { g.svc.BuySeedOrUnlock(p) } }(pt)})

		if st.HasSelectedSeed && st.SelectedSeed == pt {
			g.buttons = append(g.buttons, Button{Rect: image.Rect(hx+selectW+g.btnPad+buyW+g.btnPad, y, hx+selectW+g.btnPad+buyW+g.btnPad+statusW, y+g.btnH), Label: "[выбрано]", Enabled: false})
		}
		y += g.btnH + g.btnPad
	}
}

func (g *Game) actionStartY() int {
	return g.margin + 116
}

func (g *Game) seedsStartY() int {
	y := g.actionStartY()
	y += 5 * (g.btnH + g.btnPad)
	y += g.btnPad
	y += 3 * (g.btnH + g.btnPad)
	y += g.btnPad
	return y
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
		if v > 0 {
			return true
		}
	}
	return false
}
