package crafting

import (
	"math"

	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

type RefineOperation struct {
	Operation
	Input  materials.MaterialID
	Output []materials.Composite
}

func (d RefineOperation) GetOperation() Operation { return d.Operation }

func (d RefineOperation) Kind() string { return "refine" }

func (d RefineOperation) Execute(m *Machine, dt float32) {
	m.finishRefination(d, dt)
}

func (m *Machine) finishRefination(d RefineOperation, dt float32) {
	material, _ := materials.GetMaterial(materials.MaterialID(d.Input))
	q, ok := m.Supply.Materials.RemoveItem(container.ItemID(material.ID), 1)
	if q == 0 || !ok {
		return
	}

	substances := material.Reduction(1)
	for id, quantity := range substances {
		substance, _ := materials.GetSubstance(id)
		elements := substance.Reduce(quantity)
		for id, quantity := range elements {
			element, _ := materials.GetElement(id)
			if m.Heat >= element.BoilingPoint {
				currentLoss := (1.0 - float32(math.Exp(-0.5*float64(m.Heat-element.BoilingPoint)))) * element.Volatility * 100.0 * dt
				Consume(m, materials.ElementID(id), currentLoss)
			} else {
				Produce(m, materials.ElementID(id), quantity)
			}
		}
	}
}
