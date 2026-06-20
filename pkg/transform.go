package crafting

import (
	"math"

	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

type TransformOperation struct {
	Operation
	Input      materials.MaterialID
	Output     materials.MaterialID
	Amount     float32
	Multiplier float32
}

func (s TransformOperation) GetOperation() Operation { return s.Operation }
func (s TransformOperation) Kind() string            { return "transform" }

func (t TransformOperation) Execute(m *Machine, dt float32) {
	m.finishTransformation(t, dt)
}

func (m *Machine) finishTransformation(t TransformOperation, dt float32) {
	targetAmount := t.Amount
	if targetAmount <= 0 {
		targetAmount = 1.0
	}

	inputID := container.ItemID(t.Input)
	available := m.Supply.Materials.Items[inputID]
	if available < targetAmount {
		m.Progress = 0
		return
	}

	if t.Operation.Duration <= 0 {
		m.executeTransformationSlice(t, targetAmount)
		return
	}

	m.Progress += dt

	if m.Progress >= t.Operation.Duration {
		cycles := float32(math.Floor(float64(m.Progress / t.Operation.Duration)))
		totalToProcess := targetAmount * cycles

		if available < totalToProcess {
			totalToProcess = float32(math.Floor(float64(available/targetAmount))) * targetAmount
		}

		if totalToProcess > 0 {
			m.executeTransformationSlice(t, totalToProcess)
		}

		m.Progress = float32(math.Mod(float64(m.Progress), float64(t.Operation.Duration)))
	}
}

func (m *Machine) executeTransformationSlice(t TransformOperation, amount float32) {
	removed, ok := m.Supply.Materials.RemoveItem(container.ItemID(t.Input), amount)
	if !ok || removed <= 0 {
		return
	}

	origMaterial, err := materials.GetMaterial(t.Output)
	if err != nil {
		m.Supply.Materials.AddItem(container.ItemID(t.Input), removed)
		return
	}

	transformedInstance := origMaterial
	transformedInstance.Temperature = m.Heat

	m.Supply.Materials.AddItem(container.ItemID(transformedInstance.ID), amount*t.Multiplier)
}
