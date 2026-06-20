package crafting

import (
	"fmt"

	energy "github.com/GoGamesLab/Energy/pkg"
	container "github.com/GoGamesLab/Inventory/pkg"
)

var (
	Machines             = make(map[MachineID]Machine)
	RefineOperations     = make(map[OperationID]RefineOperation)
	SynthesizeOperations = make(map[OperationID]SynthesizeOperation)
	TransformOperations  = make(map[OperationID]TransformOperation)
)

func RegisterMachine(m Machine) error {
	if _, exists := Machines[m.ID]; exists {
		return fmt.Errorf("🧨 Máquina com ID %s já registrada", m.ID)
	}

	m.Supply = &Supply{
		Materials:  *container.NewStorage(),
		Substances: *container.NewStorage(),
		Elements:   *container.NewStorage(),
	}
	Machines[m.ID] = m

	return nil
}

func (m Machine) SetupEnergyManager() {
	m.EnergyManager = energy.NewEnergyManager()
}

func registerOp[T Procedure](store map[OperationID]T, op T) error {
	id := op.GetOperation().ID
	if _, exists := store[id]; exists {
		return fmt.Errorf("🧨 Operação com ID %s já registrada", id)
	}
	store[id] = op
	return nil
}

func RegisterOperation(p Procedure) error {
	switch op := p.(type) {
	case RefineOperation:
		return registerOp(RefineOperations, op)
	case SynthesizeOperation:
		return registerOp(SynthesizeOperations, op)
	case TransformOperation:
		return registerOp(TransformOperations, op)
	default:
		return fmt.Errorf("🧨 Tipo de operação desconhecida: %T", p)
	}
}

func GetMachine(id MachineID) *Machine {
	if m, ok := Machines[id]; ok {
		return &m
	}

	m := Machines["entropy"]
	return &m
}
