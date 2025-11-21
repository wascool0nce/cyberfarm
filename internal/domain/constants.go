package domain

import "time"

const (
	FarmWidth     = 10
	FarmHeight    = 10
	TotalPlots    = FarmWidth * FarmHeight
	SpoilDuration = 15 * time.Second
)

// PlantType enumerates all crops.
type PlantType int

const (
	PlantTomato PlantType = iota
	PlantCucumber
	PlantPotato
	PlantCarrot
	PlantTurnip
	PlantCabbage
	PlantBeet
	PlantSunflower
	PlantTypeCount
)

var PlantNames = map[PlantType]string{
	PlantTomato:    "Помидор",
	PlantCucumber:  "Огурец",
	PlantPotato:    "Картофель",
	PlantCarrot:    "Морковь",
	PlantTurnip:    "Брюква",
	PlantCabbage:   "Капуста",
	PlantBeet:      "Свекла",
	PlantSunflower: "Подсолнух",
}

// Growth time per crop.
var GrowthTimes = map[PlantType]time.Duration{
	PlantCucumber:  10 * time.Second,
	PlantTomato:    5 * time.Second,
	PlantTurnip:    15 * time.Second,
	PlantPotato:    8 * time.Second,
	PlantCarrot:    6 * time.Second,
	PlantCabbage:   12 * time.Second,
	PlantBeet:      14 * time.Second,
	PlantSunflower: 18 * time.Second,
}

// Prices for buying seeds and selling produce.
var SeedPrices = map[PlantType]int{
	PlantTomato:    1,
	PlantCucumber:  2,
	PlantPotato:    3,
	PlantCarrot:    4,
	PlantTurnip:    5,
	PlantCabbage:   7,
	PlantBeet:      8,
	PlantSunflower: 10,
}

var SellPrices = map[PlantType]int{
	PlantTomato:    3,
	PlantCucumber:  4,
	PlantPotato:    6,
	PlantCarrot:    7,
	PlantTurnip:    9,
	PlantCabbage:   12,
	PlantBeet:      14,
	PlantSunflower: 18,
}

func ExpandPlotCost(index int) int { // index == current available plots count
	return 5 + index
}

// Seed unlock costs (0 means unlocked from start).
var SeedUnlockCosts = map[PlantType]int{
	PlantTomato:    0,
	PlantCucumber:  0,
	PlantPotato:    10,
	PlantCarrot:    15,
	PlantTurnip:    22,
	PlantCabbage:   30,
	PlantBeet:      38,
	PlantSunflower: 48,
}

// Score coefficients by action key.
var ScoreCoef = map[string]float64{
	"plant":   0.5,
	"water":   0.3,
	"harvest": 1.2,
	"sell":    0.4,
	"buySeed": 0.1,
	"expand":  2,
	"upgrade": 3,
}
