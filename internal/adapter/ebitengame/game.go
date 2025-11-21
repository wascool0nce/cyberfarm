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

	tile      int
	margin    int
	midWidth  int
	shopWidth int
	columnGap int
	columnPad int

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
		tile:          36,
		margin:        18,
		midWidth:      360,
		shopWidth:     280,
		columnGap:     28,
		columnPad:     16,
		btnW:          320,
		btnH:          38,
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
	screen.Fill(color.RGBA{0x04, 0x03, 0x0c, 0xff})
	glowLayer := uint8(22 + 16*math.Sin(g.animPhase*1.4))
	ebitenutil.DrawRect(screen, 0, 0, float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy()), color.RGBA{0x07, 0x09, 0x18, glowLayer})

	ox, oy := g.margin, g.columnTop()
	for i := 0; i < domain.TotalPlots; i++ {
		x := i % domain.FarmWidth
		y := i / domain.FarmWidth
		px := ox + x*(g.tile+2)
		py := oy + y*(g.tile+2)

		pulse := 0.08 + 0.06*math.Sin(g.animPhase*2+float64(x+y))
		c := color.RGBA{uint8(float64(0x2a) + 70*pulse), uint8(float64(0x03) + 10*pulse), uint8(float64(0x18) + 22*pulse), 0xff}
		border := color.RGBA{0xff, 0x1b, 0xa5, 0xff}
		if i >= st.AvailablePlots || !st.Plots[i].Bought {
			c = color.RGBA{0x14, 0x10, 0x22, 0xff}
			border = color.RGBA{0x35, 0x2f, 0x49, 0xff}
		}
		if g.hoveredPlot == i {
			c = color.RGBA{0x1b, 0x18, 0x30, 0xff}
			border = color.RGBA{0x4a, 0xff, 0xf2, 0xff}
		}
		ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), float64(g.tile), c)
		ebitenutil.DrawRect(screen, float64(px), float64(py), float64(g.tile), 2, border)
		ebitenutil.DrawRect(screen, float64(px), float64(py+g.tile-2), float64(g.tile), 2, border)
		ebitenutil.DrawRect(screen, float64(px), float64(py), 2, float64(g.tile), border)
		ebitenutil.DrawRect(screen, float64(px+g.tile-2), float64(py), 2, float64(g.tile), border)

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

	panelHeight := screen.Bounds().Dy() - g.columnTop() - g.margin
	panelBorder := color.RGBA{0xff, 0x2b, 0xb8, 0xc0}
	panelFill := color.RGBA{0x0c, 0x0a, 0x1a, 0xa8}
	drawPanel(screen, g.midColumnX(), g.columnTop()-10, g.midWidth, panelHeight+20, panelFill, panelBorder)
	drawPanel(screen, g.shopColumnX(), g.columnTop()-10, g.shopWidth, panelHeight+20, panelFill, panelBorder)

	sw := screen.Bounds().Dx()
	title := "Киберпанк Ферма Симулятор"
	tw := text.BoundString(g.faces.Large, title).Dx()
	drawNeonText(screen, title, g.faces.Large, sw/2-tw/2, g.margin+22, color.RGBA{0xff, 0x2e, 0xa8, 0xff})

	midX := g.midColumnX() + g.columnPad
	shopX := g.shopColumnX() + g.columnPad
	colTop := g.columnTop()
	white := color.RGBA{0xe8, 0xff, 0xff, 0xff}
	accent := color.RGBA{0x82, 0xff, 0xf9, 0xff}
	muted := color.RGBA{0xad, 0xb6, 0xc5, 0xff}

	drawNeonText(screen, "Управление фермой", g.faces.Large, midX, colTop+6, accent)
	drawNeonText(screen, fmt.Sprintf("Дублоны: %d", st.Coins), g.faces.Normal, midX, colTop+44, white)
	drawNeonText(screen, fmt.Sprintf("Счёт: %.1f", st.TotalScore), g.faces.Normal, midX, colTop+68, white)

	text.Draw(screen, "Выберите клетку поля", g.faces.Normal, midX, colTop+98, muted)
	text.Draw(screen, "и действие:", g.faces.Normal, midX, colTop+120, muted)

	for i, b := range g.buttons {
		g.drawButton(screen, b, i == g.hoveredButton)
	}

	selY := g.selectionStartY()
	drawNeonText(screen, "Выберите семя для посадки:", g.faces.Normal, midX, selY-g.btnPad*2, white)

	infoY := g.infoStartY()
	drawNeonText(screen, "Состояние поля", g.faces.Normal, midX, infoY, accent)
	infoY += 22
	if st.SelectedPlot >= 0 && st.SelectedPlot < len(st.Plots) {
		p := st.Plots[st.SelectedPlot]
		text.Draw(screen, fmt.Sprintf("Клетка #%d", st.SelectedPlot+1), g.faces.Small, midX, infoY, white)
		infoY += 18
		text.Draw(screen, fmt.Sprintf("Куплена: %s", yesNo(p.Bought)), g.faces.Small, midX, infoY, white)
		infoY += 18
		text.Draw(screen, fmt.Sprintf("Растение: %s", plantNameOrDash(p)), g.faces.Small, midX, infoY, white)
		infoY += 18
		text.Draw(screen, fmt.Sprintf("Полита: %s", yesNo(p.Watered)), g.faces.Small, midX, infoY, white)
		infoY += 18
		text.Draw(screen, fmt.Sprintf("Готово к сбору: %s", yesNo(p.Harvestable)), g.faces.Small, midX, infoY, white)
		infoY += 22
	} else {
		text.Draw(screen, "Клетка не выбрана", g.faces.Small, midX, infoY, white)
		infoY += 22
	}

	drawNeonText(screen, "Инвентарь", g.faces.Normal, midX, infoY, accent)
	infoY += 20
	text.Draw(screen, fmt.Sprintf("Семена: помидор %d | огурец %d", st.Seeds[domain.PlantTomato], st.Seeds[domain.PlantCucumber]), g.faces.Small, midX, infoY, white)
	infoY += 18
	text.Draw(screen, fmt.Sprintf("картофель %d | морковь %d", st.Seeds[domain.PlantPotato], st.Seeds[domain.PlantCarrot]), g.faces.Small, midX, infoY, white)
	infoY += 18
	text.Draw(screen, fmt.Sprintf("брюква %d | капуста %d", st.Seeds[domain.PlantTurnip], st.Seeds[domain.PlantCabbage]), g.faces.Small, midX, infoY, white)
	infoY += 18
	text.Draw(screen, fmt.Sprintf("свекла %d | подсолнух %d", st.Seeds[domain.PlantBeet], st.Seeds[domain.PlantSunflower]), g.faces.Small, midX, infoY, white)
	infoY += 18
	text.Draw(screen, fmt.Sprintf("Урожай: %v", st.Produce), g.faces.Small, midX, infoY, white)
	infoY += 24

	logY := g.logStartY()
	drawNeonText(screen, "Журнал событий", g.faces.Normal, midX, logY, accent)
	logY += 22
	max := g.logLines()
	for i := 0; i < len(st.Log) && i < max; i++ {
		text.Draw(screen, st.Log[i], g.faces.Small, midX, logY, white)
		logY += 16
	}

	drawNeonText(screen, "Магазин семян", g.faces.Large, shopX, colTop+6, color.RGBA{0xff, 0x2e, 0xa8, 0xff})
	text.Draw(screen, "Покупайте или открывайте новые культуры", g.faces.Small, shopX, colTop+34, muted)
	shopY := g.shopListStartY()
	for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
		rowY := shopY + int(pt)*(g.btnH+g.btnPad)
		status := fmt.Sprintf("Семена %s", domain.PlantNames[pt])
		text.Draw(screen, status, g.faces.Normal, shopX, rowY+14, white)
		var line string
		if !st.UnlockedSeeds[pt] {
			line = fmt.Sprintf("Открыть за %d дуб., есть: %d", domain.SeedUnlockCosts[pt], st.Seeds[pt])
		} else {
			line = fmt.Sprintf("Цена: %d дуб., запас: %d", domain.SeedPrices[pt], st.Seeds[pt])
		}
		text.Draw(screen, line, g.faces.Small, shopX, rowY+34, muted)
		if st.HasSelectedSeed && st.SelectedSeed == pt {
			text.Draw(screen, "▶ выбрано", g.faces.Small, shopX+170, rowY+22, accent)
		}
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
	totalW := g.margin + g.gridWidth() + g.columnGap + g.midWidth + g.columnGap + g.shopWidth + g.margin
	height := g.logStartY() + g.logLines()*16 + g.margin + 30
	gridBottom := g.columnTop() + g.gridHeight() + g.margin
	if gridBottom > height {
		height = gridBottom
	}
	return totalW, height
}

func (g *Game) gridWidth() int {
	return domain.FarmWidth * (g.tile + 2)
}

func (g *Game) gridHeight() int {
	return domain.FarmHeight * (g.tile + 2)
}

func (g *Game) headerHeight() int {
	return 56
}

func (g *Game) columnTop() int {
	return g.margin + g.headerHeight()
}

func (g *Game) actionCount() int {
	return 5
}

func (g *Game) upgradeCount() int {
	return 3
}

func (g *Game) actionsStartY() int {
	return g.columnTop() + 146
}

func (g *Game) upgradesStartY() int {
	return g.actionsStartY() + g.actionCount()*(g.btnH+g.btnPad) + g.btnPad*2
}

func (g *Game) selectionRows() int {
	return (int(domain.PlantTypeCount) + 1) / 2
}

func (g *Game) selectionStartY() int {
	return g.upgradesStartY() + g.upgradeCount()*(g.btnH+g.btnPad) + g.btnPad*3
}

func (g *Game) infoStartY() int {
	return g.selectionStartY() + g.selectionRows()*(g.btnH+g.btnPad) + g.btnPad*3
}

func (g *Game) infoBlockHeight() int {
	return 190
}

func (g *Game) logStartY() int {
	return g.infoStartY() + g.infoBlockHeight()
}

func (g *Game) logLines() int {
	return 9
}

func (g *Game) midColumnX() int {
	return g.margin + g.gridWidth() + g.columnGap
}

func (g *Game) shopColumnX() int {
	return g.midColumnX() + g.midWidth + g.columnGap
}

func (g *Game) shopListStartY() int {
	return g.columnTop() + 66
}

func drawNeonText(screen *ebiten.Image, str string, face font.Face, x, y int, main color.Color) {
	shadow := color.RGBA{0x00, 0xff, 0xff, 0x60}
	accent := color.RGBA{0xff, 0x2d, 0x95, 0x70}
	text.Draw(screen, str, face, x-1, y-1, shadow)
	text.Draw(screen, str, face, x+1, y+1, accent)
	text.Draw(screen, str, face, x, y, main)
}

func drawPanel(screen *ebiten.Image, x, y, w, h int, fill color.Color, border color.Color) {
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(w), float64(h), fill)
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(w), 2, border)
	ebitenutil.DrawRect(screen, float64(x), float64(y+h-2), float64(w), 2, border)
	ebitenutil.DrawRect(screen, float64(x), float64(y), 2, float64(h), border)
	ebitenutil.DrawRect(screen, float64(x+w-2), float64(y), 2, float64(h), border)
}

func plantColor(pt domain.PlantType) color.RGBA {
	switch pt {
	case domain.PlantTomato:
		return color.RGBA{0xff, 0x3c, 0x4d, 0xff}
	case domain.PlantCucumber:
		return color.RGBA{0x00, 0xf5, 0xb5, 0xff}
	case domain.PlantPotato:
		return color.RGBA{0xd2, 0x9b, 0x52, 0xff}
	case domain.PlantCarrot:
		return color.RGBA{0xff, 0x9a, 0x2d, 0xff}
	case domain.PlantTurnip:
		return color.RGBA{0xb9, 0x7b, 0xff, 0xff}
	case domain.PlantCabbage:
		return color.RGBA{0x4d, 0xff, 0x7a, 0xff}
	case domain.PlantBeet:
		return color.RGBA{0xff, 0x2b, 0x7a, 0xff}
	case domain.PlantSunflower:
		return color.RGBA{0xff, 0xd8, 0x3d, 0xff}
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

func plantNameOrDash(p domain.Plot) string {
	if p.HasPlant {
		return domain.PlantNames[p.Plant]
	}
	return "-"
}

// ---- UI helpers (mouse buttons, layout, text) ----

func (g *Game) drawButton(screen *ebiten.Image, b Button, hovered bool) {
	col := color.RGBA{0x0f, 0x0f, 0x1d, 0xff}
	border := color.RGBA{0xff, 0x2b, 0xb8, 0xff}
	txt := color.RGBA{0xe8, 0xff, 0xff, 0xff}
	if !b.Enabled {
		col = color.RGBA{0x25, 0x23, 0x36, 0xff}
		border = color.RGBA{0x44, 0x3f, 0x5a, 0xff}
		txt = color.RGBA{0x9a, 0x9c, 0xa8, 0xff}
	}
	if hovered && b.Enabled {
		boost := uint8(18 + 12*math.Sin(g.animPhase*5))
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
	ox, oy := g.margin, g.columnTop()
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

	actionX := g.midColumnX() + g.columnPad
	btnW := g.midWidth - g.columnPad*2
	y := g.actionsStartY()

	add := func(label string, enabled bool, cb func()) {
		g.buttons = append(g.buttons, Button{Rect: image.Rect(actionX, y, actionX+btnW, y+g.btnH), Label: label, Enabled: enabled, OnClick: cb})
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
	y = g.upgradesStartY()
	add(fmt.Sprintf("Засев x4 (50) [%s]", yesNo(st.CanPlantBulk)), !st.CanPlantBulk && st.Coins >= 50, func() { g.svc.BuyUpgradePlantBulk() })
	add(fmt.Sprintf("Полив x4 (30) [%s]", yesNo(st.CanWaterBulk)), !st.CanWaterBulk && st.Coins >= 30, func() { g.svc.BuyUpgradeWaterBulk() })
	add(fmt.Sprintf("Сбор x4 (40) [%s]", yesNo(st.CanHarvestBulk)), !st.CanHarvestBulk && st.Coins >= 40, func() { g.svc.BuyUpgradeHarvestBulk() })

	// seed selection grid
	selX := g.midColumnX() + g.columnPad
	selY := g.selectionStartY()
	selW := (g.midWidth - g.columnPad*3) / 2
	for idx, pt := 0, domain.PlantType(0); pt < domain.PlantTypeCount; idx, pt = idx+1, pt+1 {
		row := idx / 2
		col := idx % 2
		x := selX + col*(selW+g.columnPad)
		y := selY + row*(g.btnH+g.btnPad)
		label := domain.PlantNames[pt]
		if st.HasSelectedSeed && st.SelectedSeed == pt {
			label = "★ " + label
		}
		selectEnabled := st.UnlockedSeeds[pt] && st.Seeds[pt] > 0
		g.buttons = append(g.buttons, Button{Rect: image.Rect(x, y, x+selW, y+g.btnH), Label: label, Enabled: selectEnabled, OnClick: func(p domain.PlantType) func() { return func() { g.svc.SelectSeed(p) } }(pt)})
	}

	// shop buttons
	shopBtnW := 120
	shopY := g.shopListStartY()
	shopX := g.shopColumnX() + g.shopWidth - g.columnPad - shopBtnW
	for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
		y := shopY + int(pt)*(g.btnH+g.btnPad)
		var label string
		var enabled bool
		if !st.UnlockedSeeds[pt] {
			label = fmt.Sprintf("Открыть (%d)", domain.SeedUnlockCosts[pt])
			enabled = st.Coins >= domain.SeedUnlockCosts[pt]
		} else {
			label = fmt.Sprintf("Купить (%d)", domain.SeedPrices[pt])
			enabled = st.Coins >= domain.SeedPrices[pt]
		}
		g.buttons = append(g.buttons, Button{Rect: image.Rect(shopX, y, shopX+shopBtnW, y+g.btnH), Label: label, Enabled: enabled, OnClick: func(p domain.PlantType) func() { return func() { g.svc.BuySeedOrUnlock(p) } }(pt)})
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
		if v > 0 {
			return true
		}
	}
	return false
}
