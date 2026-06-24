package crafting

import (
	"encoding/json"
	"fmt"
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
	Converters   []struct {
		ID       string `json:"ID"`
		Quantity int    `json:"Quantity"`
	} `json:"Converters"`
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
	Converters    []energy.Converter
}

type Line struct {
	Machines []*Machine
}

type Factory struct {
	Lines     []Line
	Warehouse *Supply
}

type ResourceID interface {
	materials.ElementID | materials.SubstanceID | materials.MaterialID
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
		for _, c := range mJSON.Converters {
			m.Converters = append(m.Converters, energy.Converter{
				ID:       energy.ConverterID(c.ID),
				Quantity: c.Quantity,
			})
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
