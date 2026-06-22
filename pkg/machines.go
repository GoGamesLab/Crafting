package crafting

import (
	"fmt"
	"log"
	"math"

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

func (m *Machine) Update(dt float32) {
	var to energy.EnergyUnit

	if m.EnergyManager == nil {
		log.Fatalf("💣 no energy manager for %s machine ", m.Name)
	}

	for _, c := range m.EnergyManager.EnergyConverters {
		to = c.FromBase(1)
		if _, err := c.ToBase(to); err != nil {
			log.Fatalf("🧨 converter: %s", err)
		}
	}

	for _, step := range m.Procedures {
		p := step.GetOperation()

		if m.Heat >= p.ActivationTemp {
			m.EnergyManager.Update()

			m.executeStep(step, dt)

			m.Heat = calculateTemperature(m.Heat, p.BaseEnergyChange*dt, p.Dissipation)
		} else {
			m.Heat += p.BaseEnergyChange * dt
		}
	}
}

func (m *Machine) GetSupply() Supply {
	return Supply{
		Materials:  m.Supply.Materials,
		Substances: m.Supply.Substances,
		Elements:   m.Supply.Elements,
	}
}

func (m *Machine) Type() string { return m.TypeHint }

func (m *Machine) GenerateEnergy(amount energy.EnergyUnit) energy.EnergyUnit {
	var c energy.EnergyUnit = m.EnergyCapacity()
	if amount > c {
		return c
	}

	return amount
}

func (m *Machine) ConsumeEnergy() energy.EnergyUnit { return m.EnergyDemand() }

func (m *Machine) EnergyCapacity() energy.EnergyUnit {
	var c energy.EnergyUnit = 0
	for _, p := range m.Procedures {
		c += energy.EnergyUnit(p.GetOperation().BaseEnergyChange)
	}

	return c
}

func (m *Machine) EnergyDemand() energy.EnergyUnit {
	var t energy.EnergyUnit = 0
	for _, p := range m.Procedures {
		t += p.GetOperation().EnergyCost
	}

	return t
}

func (m *Machine) SetupEnergyManager() { m.EnergyManager = energy.NewEnergyManager() }

func (m *Machine) SetEnergyManager(e *energy.EnergyManager) { m.EnergyManager = e }

func GetMachine(id MachineID) *Machine {
	if m, ok := Machines[id]; ok {
		return &m
	}

	m := Machines["entropy"]
	return &m
}

func (m *Machine) executeStep(step Procedure, dt float32) {
	p := step.GetOperation()

	m.Progress += dt

	if m.Progress >= p.Duration {
		step.Execute(m, dt)

		m.Progress = 0
	}
}

func registerOp[T Procedure](store map[OperationID]T, op T) error {
	id := op.GetOperation().ID
	if _, exists := store[id]; exists {
		return fmt.Errorf("🧨 Operação com ID %s já registrada", id)
	}
	store[id] = op
	return nil
}

func calculateTemperature(current, heatGain float32, dissipation float32) float32 {
	const envTemperature = 21.0

	retain := math.Pow(2.0, -1.0/float64(dissipation))
	k := 1.0 - retain

	delta := float64(current - envTemperature)
	delta = delta*(1.0-k) + float64(heatGain)

	return float32(envTemperature + delta)
}
