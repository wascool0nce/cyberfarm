package usecase

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"cyberfarm/internal/domain"
)

type Service struct {
	s   *domain.GameState
	rng *rand.Rand
	// ticking
	lastTick time.Time
}

func NewService() *Service {
	svc := &Service{
		s: &domain.GameState{
			Plots:          make([]domain.Plot, domain.TotalPlots),
			AvailablePlots: 2,
			Produce:        make(domain.Inventory),
			Seeds:          make(domain.Inventory),
			Coins:          8,
			SelectedPlot:   -1,
			UnlockedSeeds:  map[domain.PlantType]bool{},
			ScoreCounters:  map[string]int{},
			Log:            make([]string, 0, 32),
		},
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		lastTick: time.Now(),
	}

	// init unlocked seeds
	for pt := domain.PlantType(0); pt < domain.PlantTypeCount; pt++ {
		svc.s.UnlockedSeeds[pt] = domain.SeedUnlockCosts[pt] == 0
	}

	// init plots
	for i := range svc.s.Plots {
		svc.s.Plots[i].Bought = i < svc.s.AvailablePlots
	}

	// init scores
	for k := range domain.ScoreCoef {
		svc.s.ScoreCounters[k] = 0
	}

	return svc
}

func (svc *Service) State() *domain.GameState { return svc.s }

// MaybeTick should be called each frame; it performs growth logic once per second.
func (svc *Service) MaybeTick(now time.Time) {
	if svc.s.Won || svc.s.PausedForEvent {
		svc.lastTick = now
		return
	}
	if now.Sub(svc.lastTick) < time.Second {
		return
	}
	svc.lastTick = now

	// 1% chance to trigger a story event
	if svc.rng.Float64() < 0.01 {
		svc.triggerEvent()
		return
	}

	updated := false
	for i := range svc.s.Plots {
		p := &svc.s.Plots[i]
		if !p.HasPlant {
			continue
		}
		if p.Harvestable {
			// spoil check
			if !p.HarvestAt.IsZero() && now.Sub(p.HarvestAt) >= domain.SpoilDuration {
				svc.logf("Растение в клетке #%d (%s) испортилось и исчезло.", i+1, svc.plantName(p.Plant))
				*p = domain.Plot{Bought: p.Bought}
				updated = true
			}
			continue
		}
		if !p.Watered {
			continue
		}
		// advance growth
		total := domain.GrowthTimes[p.Plant]
		elapsed := now.Sub(p.PlantedAt)
		stage := int(math.Min(4, math.Floor(float64(elapsed)/float64(total/5))))
		if stage != p.GrowthStage {
			p.GrowthStage = stage
			updated = true
		}
		if elapsed >= total {
			p.Harvestable = true
			p.HarvestAt = now
			svc.logf("Растение в клетке #%d (%s) созрело!", i+1, svc.plantName(p.Plant))
			updated = true
		}
	}

	if updated {
		// noop: UI reads from state each frame
	}

	if svc.s.Coins >= 1000 {
		svc.s.Won = true
	}
}

func (svc *Service) triggerEvent() {
	evs := defaultEvents()
	ev := evs[svc.rng.Intn(len(evs))]
	svc.s.CurrentEvent = &ev
	svc.s.PausedForEvent = true
}

func defaultEvents() []domain.StoryEvent {
	return []domain.StoryEvent{
		{
			ID:      "abandoned_shop",
			Message: "Вы наткнулись на заброшенный магазин с семенами. Выберите: 1) Взять помидоры (+5) 2) Взять огурцы (+5)",
			Choices: [2]domain.StoryChoice{
				{ID: "tomato", Text: "Взять помидоры", Effect: domain.EffectAddSeeds, Plant: domain.PlantTomato, Amount: 5},
				{ID: "cucumber", Text: "Взять огурцы", Effect: domain.EffectAddSeeds, Plant: domain.PlantCucumber, Amount: 5},
			},
		},
		{
			ID:      "bandits",
			Message: "Бандиты требуют часть урожая. 1) Отдать половину 2) Сбежать (50/50)",
			Choices: [2]domain.StoryChoice{
				{ID: "half", Text: "Отдать половину", Effect: domain.EffectHalveProduce},
				{ID: "escape", Text: "Попытаться сбежать", Effect: domain.EffectRiskLoseAllOrNone},
			},
		},
	}
}

func (svc *Service) ResolveEvent(choiceIdx int) {
	if !svc.s.PausedForEvent || svc.s.CurrentEvent == nil || choiceIdx < 0 || choiceIdx > 1 {
		return
	}
	ch := svc.s.CurrentEvent.Choices[choiceIdx]
	switch ch.Effect {
	case domain.EffectAddSeeds:
		svc.s.Seeds[ch.Plant] += ch.Amount
		svc.logf("Вы получили %d семян: %s.", ch.Amount, svc.plantName(ch.Plant))
	case domain.EffectHalveProduce:
		for k := range svc.s.Produce {
			svc.s.Produce[k] = svc.s.Produce[k] / 2
		}
		svc.log("Вы отдали половину урожая.")
	case domain.EffectRiskLoseAllOrNone:
		if svc.rng.Float64() < 0.5 {
			for k := range svc.s.Produce {
				svc.s.Produce[k] = 0
			}
			svc.log("Вас поймали — весь урожай потерян.")
		} else {
			svc.log("Удалось сбежать без потерь.")
		}
	}
	svc.s.CurrentEvent = nil
	svc.s.PausedForEvent = false
}

// Actions

func (svc *Service) SelectPlot(idx int) {
	if idx < 0 || idx >= len(svc.s.Plots) {
		svc.s.SelectedPlot = -1
		return
	}
	if !svc.s.Plots[idx].Bought {
		svc.s.SelectedPlot = -1
		return
	}
	if svc.s.SelectedPlot == idx {
		svc.s.SelectedPlot = -1
		return
	}
	svc.s.SelectedPlot = idx
}

func (svc *Service) SelectSeed(pt domain.PlantType) {
	if !svc.s.UnlockedSeeds[pt] {
		svc.logf("Семена %s ещё не открыты.", svc.plantName(pt))
		return
	}
	if svc.s.Seeds[pt] == 0 {
		return
	}
	if svc.s.HasSelectedSeed && svc.s.SelectedSeed == pt {
		svc.s.HasSelectedSeed = false
		return
	}
	svc.s.SelectedSeed = pt
	svc.s.HasSelectedSeed = true
}

func (svc *Service) BuySeedOrUnlock(pt domain.PlantType) {
	if !svc.s.UnlockedSeeds[pt] {
		cost := domain.SeedUnlockCosts[pt]
		if svc.s.Coins < cost {
			svc.logf("Нужно %d дублонов, чтобы открыть %s.", cost, svc.plantName(pt))
			return
		}
		svc.s.Coins -= cost
		svc.s.UnlockedSeeds[pt] = true
		svc.logf("Открыты семена: %s.", svc.plantName(pt))
		return
	}
	price := domain.SeedPrices[pt]
	if svc.s.Coins < price {
		svc.logf("Недостаточно дублонов для семян %s.", svc.plantName(pt))
		return
	}
	svc.s.Coins -= price
	svc.s.Seeds[pt]++
	svc.award("buySeed", 1)
	svc.logf("Куплены семена: %s.", svc.plantName(pt))
}

func (svc *Service) Plant() {
	if !svc.s.HasSelectedSeed {
		return
	}
	if svc.s.CanPlantBulk {
		empty := make([]int, 0, 16)
		for i := range svc.s.Plots {
			p := &svc.s.Plots[i]
			if p.Bought && !p.HasPlant {
				empty = append(empty, i)
			}
		}
		count := min(4, len(empty), svc.s.Seeds[svc.s.SelectedSeed])
		planted := 0
		for i := 0; i < count; i++ {
			idx := empty[i]
			svc.plantAt(idx, svc.s.SelectedSeed)
			planted++
		}
		if planted > 0 {
			svc.award("plant", planted)
		}
		return
	}
	idx := svc.s.SelectedPlot
	if idx < 0 {
		return
	}
	svc.plantAt(idx, svc.s.SelectedSeed)
	svc.award("plant", 1)
}

func (svc *Service) plantAt(idx int, pt domain.PlantType) {
	p := &svc.s.Plots[idx]
	if !p.Bought || p.HasPlant {
		return
	}
	if svc.s.Seeds[pt] <= 0 {
		svc.logf("Нет семян %s.", svc.plantName(pt))
		return
	}
	*p = domain.Plot{
		Bought:      true,
		HasPlant:    true,
		Plant:       pt,
		PlantedAt:   time.Now(),
		Watered:     false,
		GrowthStage: 0,
	}
	svc.s.Seeds[pt]--
	svc.logf("Посадили %s в клетку #%d.", svc.plantName(pt), idx+1)
}

func (svc *Service) Water() {
	if svc.s.CanWaterBulk {
		indices := make([]int, 0, 16)
		for i := range svc.s.Plots {
			p := svc.s.Plots[i]
			if p.Bought && p.HasPlant && !p.Watered && !p.Harvestable {
				indices = append(indices, i)
			}
		}
		cnt := min(4, len(indices))
		for i := 0; i < cnt; i++ {
			idx := indices[i]
			svc.s.Plots[idx].Watered = true
			svc.logf("Полили растение в клетке #%d.", idx+1)
		}
		if cnt > 0 {
			svc.award("water", cnt)
		}
		return
	}
	idx := svc.s.SelectedPlot
	if idx < 0 {
		return
	}
	p := &svc.s.Plots[idx]
	if !p.HasPlant || p.Watered || p.Harvestable {
		return
	}
	p.Watered = true
	svc.award("water", 1)
	svc.logf("Полили растение в клетке #%d.", idx+1)
}

func (svc *Service) Harvest() {
	if svc.s.CanHarvestBulk {
		indices := make([]int, 0, 16)
		for i := range svc.s.Plots {
			p := svc.s.Plots[i]
			if p.Bought && p.Harvestable {
				indices = append(indices, i)
			}
		}
		cnt := min(4, len(indices))
		for i := 0; i < cnt; i++ {
			svc.harvestAt(indices[i])
		}
		if cnt > 0 {
			svc.award("harvest", cnt)
		}
		return
	}
	idx := svc.s.SelectedPlot
	if idx < 0 {
		return
	}
	if svc.harvestAt(idx) {
		svc.award("harvest", 1)
	}
}

func (svc *Service) harvestAt(idx int) bool {
	p := &svc.s.Plots[idx]
	if !p.Harvestable || !p.HasPlant {
		return false
	}
	svc.s.Produce[p.Plant]++
	svc.logf("Собрали урожай: %s с клетки #%d.", svc.plantName(p.Plant), idx+1)
	*p = domain.Plot{Bought: true}
	return true
}

func (svc *Service) SellAll() {
	total := 0
	soldItems := 0
	for pt, qty := range svc.s.Produce {
		if qty > 0 {
			total += qty * domain.SellPrices[pt]
			soldItems += qty
			svc.s.Produce[pt] = 0
		}
	}
	if total == 0 {
		svc.log("Нет урожая для продажи.")
		return
	}
	svc.s.Coins += total
	if soldItems > 0 {
		svc.award("sell", soldItems)
	}
	svc.logf("Продали весь урожай за %d дублонов!", total)
}

func (svc *Service) ExpandField() {
	if svc.s.AvailablePlots >= domain.TotalPlots {
		return
	}
	cost := domain.ExpandPlotCost(svc.s.AvailablePlots)
	if svc.s.Coins < cost {
		svc.log("Недостаточно дублонов для покупки клетки.")
		return
	}
	svc.s.Coins -= cost
	svc.s.Plots[svc.s.AvailablePlots].Bought = true
	svc.s.AvailablePlots++
	svc.award("expand", 1)
	svc.logf("Куплена новая клетка. Доступно: %d.", svc.s.AvailablePlots)
}

func (svc *Service) BuyUpgradePlantBulk() {
	if svc.s.CanPlantBulk {
		svc.log("Улучшение уже куплено.")
		return
	}
	if svc.s.Coins < 50 {
		svc.log("Недостаточно дублонов для улучшения засева.")
		return
	}
	svc.s.Coins -= 50
	svc.s.CanPlantBulk = true
	svc.award("upgrade", 1)
	svc.log("Куплено: улучшенный засев (до 4 клеток).")
}

func (svc *Service) BuyUpgradeWaterBulk() {
	if svc.s.CanWaterBulk {
		svc.log("Улучшение уже куплено.")
		return
	}
	if svc.s.Coins < 30 {
		svc.log("Недостаточно дублонов для улучшения.")
		return
	}
	svc.s.Coins -= 30
	svc.s.CanWaterBulk = true
	svc.award("upgrade", 1)
	svc.log("Куплено: полив 4 клеток.")
}

func (svc *Service) BuyUpgradeHarvestBulk() {
	if svc.s.CanHarvestBulk {
		svc.log("Улучшение уже куплено.")
		return
	}
	if svc.s.Coins < 40 {
		svc.log("Недостаточно дублонов для улучшения.")
		return
	}
	svc.s.Coins -= 40
	svc.s.CanHarvestBulk = true
	svc.award("upgrade", 1)
	svc.log("Куплено: сбор 4 клеток.")
}

// Helpers

func (svc *Service) plantName(pt domain.PlantType) string {
	return domain.PlantNames[pt]
}

func (svc *Service) award(action string, amount int) {
	if amount <= 0 {
		return
	}
	if _, ok := svc.s.ScoreCounters[action]; ok {
		svc.s.ScoreCounters[action] += amount
		// recompute total score
		total := 0.0
		for k, v := range svc.s.ScoreCounters {
			coef := domain.ScoreCoef[k]
			total += float64(v) * coef
		}
		svc.s.TotalScore = total
	}
}

func (svc *Service) logf(format string, args ...any) { svc.log(fmt.Sprintf(format, args...)) }

func (svc *Service) log(msg string) {
	timeStr := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", timeStr, msg)
	svc.s.Log = append([]string{entry}, svc.s.Log...)
	if len(svc.s.Log) > 20 {
		svc.s.Log = svc.s.Log[:20]
	}
}

func min(a int, b ...int) int {
	m := a
	for _, x := range b {
		if x < m {
			m = x
		}
	}
	return m
}
