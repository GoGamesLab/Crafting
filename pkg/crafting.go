package crafting

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"

	energy "github.com/GoGamesLab/Energy/pkg"
	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

type CraftDataJSON struct {
	Operations []json.RawMessage `json:"operations"`
	Machines   []MachineJSON     `json:"machines"`
}

type OperationTypeCheck struct {
	Type string `json:"Type"`
}

type MachineJSON struct {
	ID           MachineID `json:"ID"`
	Name         string    `json:"Name"`
	Heat         float32   `json:"Heat"`
	ProcedureIDs []string  `json:"ProcedureIDs"`
	TypeHint     string    `json:"TypeHint"`
}

type OperationID string
type MachineID string

type Operation struct {
	ID               OperationID
	Name             string
	RequiredTemp     float32
	EnergyCost       energy.EnergyUnit
	Duration         float32
	BaseEnergyChange float32
	ActivationTemp   float32
	Dissipation      float32
}

type Procedure interface {
	GetOperation() Operation
	Kind() string
	Execute(m *Machine, dt float32)
}

type Supply struct {
	Materials  container.Storage
	Substances container.Storage
	Elements   container.Storage
}

type Machine struct {
	ID            MachineID
	Name          string
	Heat          float32
	Procedures    []Procedure
	Progress      float32
	Supply        *Supply
	TypeHint      string
	EnergyManager *energy.EnergyManager
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

func LoadDefinitionsFromJSON(filepath string) error {
	fileData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("falha ao ler o arquivo: %w", err)
	}

	var gameData CraftDataJSON
	if err := json.Unmarshal(fileData, &gameData); err != nil {
		return fmt.Errorf("falha no unmarshal do JSON base: %w", err)
	}

	for _, rawOp := range gameData.Operations {
		var typeCheck OperationTypeCheck
		if err := json.Unmarshal(rawOp, &typeCheck); err != nil {
			return fmt.Errorf("falha ao identificar tipo da operação: %w", err)
		}

		switch typeCheck.Type {
		case "refine":
			var op RefineOperation
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return err
			}
			if err := RegisterOperation(op); err != nil {
				return err
			}

		case "synthesize":
			var op SynthesizeOperation
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return err
			}
			if err := RegisterOperation(op); err != nil {
				return err
			}

		case "transform":
			var op TransformOperation
			if err := json.Unmarshal(rawOp, &op); err != nil {
				return err
			}
			if err := RegisterOperation(op); err != nil {
				return err
			}

		default:
			return fmt.Errorf("🧨 Tipo de operação desconhecido no JSON: %s", typeCheck.Type)
		}
	}

	for _, mJSON := range gameData.Machines {
		m := Machine{
			ID:       mJSON.ID,
			Name:     mJSON.Name,
			Heat:     mJSON.Heat,
			TypeHint: mJSON.TypeHint,
		}

		for _, procID := range mJSON.ProcedureIDs {
			proc, err := findProcedure(OperationID(procID))
			if err != nil {
				return fmt.Errorf("máquina %s referenciou procedure inexistente: %w", m.ID, err)
			}
			m.Procedures = append(m.Procedures, proc)
		}

		if err := RegisterMachine(m); err != nil {
			return err
		}
	}

	return nil
}

type ResourceID interface {
	materials.ElementID | materials.SubstanceID | materials.MaterialID
}

func Store[K ResourceID](m *Machine, id K, amount float32) {
	if amount <= 0 {
		return
	}

	switch any(new(K)).(type) {
	case *materials.ElementID:
		m.Supply.Elements.AddItem(container.ItemID(any(id).(materials.ElementID)), amount)
	case *materials.SubstanceID:
		m.Supply.Substances.AddItem(container.ItemID(any(id).(materials.SubstanceID)), amount)
	case *materials.MaterialID:
		m.Supply.Materials.AddItem(container.ItemID(any(id).(materials.MaterialID)), amount)
	}
}

func (m *Machine) Type() string { return m.TypeHint }

func (m *Machine) Produce(amount energy.EnergyUnit) energy.EnergyUnit {
	var c energy.EnergyUnit = m.Capacity()
	if amount > c {
		return c
	}

	return amount
}

func (m *Machine) Consume() energy.EnergyUnit { return m.Demand() }

func (m *Machine) Capacity() energy.EnergyUnit {
	var c energy.EnergyUnit = 0
	for _, p := range m.Procedures {
		c += energy.EnergyUnit(p.GetOperation().BaseEnergyChange)
	}

	return c
}

func (m *Machine) Demand() energy.EnergyUnit {
	var t energy.EnergyUnit = 0
	for _, p := range m.Procedures {
		t += p.GetOperation().EnergyCost
	}

	return t
}

func Produce[K ResourceID](m *Machine, id K, amount float32) {
	if amount <= 0 {
		return
	}

	switch any(new(K)).(type) {
	case *materials.ElementID:
		m.Supply.Elements.AddItem(container.ItemID(any(id).(materials.ElementID)), amount/100)
	case *materials.SubstanceID:
		m.Supply.Substances.AddItem(container.ItemID(any(id).(materials.SubstanceID)), amount/100)
	case *materials.MaterialID:
		m.Supply.Materials.AddItem(container.ItemID(any(id).(materials.MaterialID)), amount/100)
	}
}

func Consume[K ResourceID](m *Machine, id K, amount float32) {
	if amount <= 0 {
		return
	}

	switch any(new(K)).(type) {
	case *materials.ElementID:
		m.Supply.Elements.RemoveItem(container.ItemID(any(id).(materials.ElementID)), amount/100)
	case *materials.SubstanceID:
		m.Supply.Substances.RemoveItem(container.ItemID(any(id).(materials.SubstanceID)), amount/100)
	case *materials.MaterialID:
		m.Supply.Materials.RemoveItem(container.ItemID(any(id).(materials.MaterialID)), amount/100)
	}
}

func (m *Machine) executeStep(step Procedure, dt float32) {
	p := step.GetOperation()

	m.Progress += dt

	if m.Progress >= p.Duration {
		step.Execute(m, dt)

		m.Progress = 0
	}
}

func calculateTemperature(current, heatGain float32, dissipation float32) float32 {
	const envTemperature = 21.0

	retain := math.Pow(2.0, -1.0/float64(dissipation))
	k := 1.0 - retain

	delta := float64(current - envTemperature)
	delta = delta*(1.0-k) + float64(heatGain)

	return float32(envTemperature + delta)
}

func findProcedure(id OperationID) (Procedure, error) {
	if op, exists := RefineOperations[id]; exists {
		return op, nil
	}
	if op, exists := SynthesizeOperations[id]; exists {
		return op, nil
	}
	if op, exists := TransformOperations[id]; exists {
		return op, nil
	}

	return nil, fmt.Errorf("operador %s não encontrado em nenhum registro", id)
}
