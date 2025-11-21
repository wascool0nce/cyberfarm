package domain

import "time"

type Plot struct {
    Bought       bool
    HasPlant     bool
    Plant        PlantType
    PlantedAt    time.Time
    Watered      bool
    GrowthStage  int // 0..4
    Harvestable  bool
    HarvestAt    time.Time
}

type Inventory map[PlantType]int

type GameState struct {
    Plots          []Plot
    AvailablePlots int

    Produce Inventory
    Seeds   Inventory

    Coins int

    SelectedPlot int // -1 if none
    SelectedSeed PlantType
    HasSelectedSeed bool

    UnlockedSeeds map[PlantType]bool

    // Upgrades
    CanPlantBulk   bool
    CanWaterBulk   bool
    CanHarvestBulk bool

    // Scoring
    ScoreCounters map[string]int
    TotalScore    float64

    // Story Events
    PausedForEvent bool
    CurrentEvent   *StoryEvent

    // Win
    Won bool

    // Log (last N entries)
    Log []string
}

type EffectType int

const (
    EffectAddSeeds EffectType = iota // params: PlantType, Amount
    EffectHalveProduce               // no params
    EffectRiskLoseAllOrNone          // probability 0.5
)

type StoryChoice struct {
    ID     string
    Text   string
    Effect EffectType
    Plant  PlantType // used by EffectAddSeeds
    Amount int       // used by EffectAddSeeds
}

type StoryEvent struct {
    ID      string
    Message string
    Choices [2]StoryChoice
}

