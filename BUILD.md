# Build & Run (Go + Ebitengine)

Prerequisites
- Go 1.22+
- Network access to fetch modules (`ebiten/v2`, `x/image`).

Commands
- `go run ./cmd/cyberfarm` — start the game.
- `go build -o bin/cyberfarm ./cmd/cyberfarm` — build a binary.

Controls
- Mouse: click a plot to select it.
- Use right-side buttons to Plant, Water, Harvest, Sell, or Expand.
- In the Seeds section: "Выбрать" selects a seed; "Купить/Открыть" purchases or unlocks it.
- Upgrades section: click to buy bulk actions.
- Story events: click one of the two choices.

Notes
- Story events pause growth; click a choice to continue.
- Win at 1000 coins; score shown in HUD.
