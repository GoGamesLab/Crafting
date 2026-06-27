package crafting

import (
	"math"

	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

type SynthesizeOperation struct {
	Operation
	Input  []materials.Composite
	Output materials.MaterialID
}

func (s SynthesizeOperation) GetOperation() Operation { return s.Operation }

func (s SynthesizeOperation) Kind() string { return "synthesize" }

func (s SynthesizeOperation) Execute(m *Machine, dt float32) { m.finishSynthesization(s, dt) }

func (m *Machine) finishSynthesization(s SynthesizeOperation, dt float32) {
	possibleMats, err := materials.Compositions(s.Input)
	if err != nil {
		return
	}
	for _, possible := range possibleMats {
		if s.Output == possible.ID {
			material, err := materials.GetMaterial(s.Output)
			if err != nil {
				return
			}
			elements := material.Reduction(dt)
			var production float64 = math.Inf(1)
			for id, needed := range elements {
				available := m.Supply.Substances.Items[container.ItemID(id)]
				amount := available / needed
				production = math.Min(float64(amount), float64(production))
				Consume(m.Supply, id, amount)
			}
			Produce(m.Supply, material.ID, float32(production)*100)

			break
		}
	}
}
