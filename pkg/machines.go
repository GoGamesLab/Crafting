package crafting

import (
	"fmt"

	energy "github.com/GoGamesLab/Energy/pkg"
	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

var (
	Machines             = make(map[MachineID]Machine)
	RefineOperations     = make(map[OperationID]RefineOperation)
	SynthesizeOperations = make(map[OperationID]SynthesizeOperation)
	TransformOperations  = make(map[OperationID]TransformOperation)
)

func init() {
	RegisterOperation(RefineOperation{
		Operation: Operation{
			ID:               "coal_combustion",
			Name:             "Combustão de Carvão",
			RequiredTemp:     600.0, // Temperatura ideal de queima
			ActivationTemp:   300.0, // FlashPoint do seu Carvão
			Duration:         10.0,  // Cada unidade de carvão dura 10 segundos
			EnergyCost:       0,     // Não gasta energia elétrica/externa
			BaseEnergyChange: 150.0, // Produz calor para a máquina e arredores
			Dissipation:      3.5,   // Perda para o ambiente (0 = 100%)
		},
		// O que a fogueira "consome" do inventário
		Input: "coal",
		// O que sobra (Cinzas/Carbono e Gases para a atmosfera) deve ser jogado
		// inventário da máquina
		Output: []materials.Composite{
			{Substance: "pure_carbon", Percentual: 5},
			{Substance: "pure_sulfur", Percentual: 5},
			{Substance: "methane", Percentual: 15},
		},
	})
	RegisterOperation(RefineOperation{
		Operation: Operation{
			ID:               "crude_oil_combustion",
			Name:             "Combustão de Óleo Crú",
			RequiredTemp:     700.0, // Temperatura ideal de queima
			ActivationTemp:   400.0, // FlashPoint do seu Óleo
			Duration:         15.0,  // Cada unidade de óleo dura 5 segundos
			EnergyCost:       0,     // Não gasta energia elétrica/externa
			BaseEnergyChange: 250.0, // Produz calor para a máquina e arredores
			Dissipation:      3.5,   // Perda para o ambiente (0 = 100%)
		},
		// O que a fogueira "consome" do mundo
		Input: "crude_oil",
		// O que sobra (Cinzas/Carbono e Gases para a atmosfera) deve ser jogado
		// inventário da máquina
		Output: []materials.Composite{
			{Substance: "paraffin", Percentual: 2},
			{Substance: "methane", Percentual: 5},
			{Substance: "pure_sulfur", Percentual: 3},
		},
	})
	RegisterOperation(TransformOperation{
		Operation: Operation{
			ID:               "manual_press_bar",
			Name:             "Prensa manual de metais em barras",
			ActivationTemp:   20.0,
			Duration:         15.0,
			EnergyCost:       500,
			BaseEnergyChange: 5.0,
			Dissipation:      0.1,
		},
		Input:  materials.MaterialID("pure_iron"),
		Output: "iron_bar",
		Amount: 1,
	})
	RegisterOperation(TransformOperation{
		Operation: Operation{
			ID:               "manual_press_plate",
			Name:             "Prensa manual de metais em placas",
			ActivationTemp:   20.0,
			Duration:         15.0,
			EnergyCost:       500,
			BaseEnergyChange: 5.0,
			Dissipation:      0.1,
		},
		Input:  "iron_bar",
		Output: "iron_plate",
		Amount: 1,
	})

	// PROCESSO 1: Lixiviação/Refino Químico (Minério -> Yellowcake)
	RegisterOperation(RefineOperation{
		Operation: Operation{
			ID:               "uranium_leaching",
			Name:             "Lixiviação Ácida de Urânio",
			RequiredTemp:     80.0, // Processo químico aquecido
			ActivationTemp:   20.0,
			Duration:         20.0,  // Demorado
			EnergyCost:       2000,  // Consome energia elétrica da máquina
			BaseEnergyChange: -10.0, // Consome calor (endotérmico)
			Dissipation:      1.0,
		},
		Input: "uranium_ore",
		Output: []materials.Composite{
			{Substance: "uranium_dioxide", Percentual: 80},
			{Substance: "pure_silicon", Percentual: 20},
		},
	})

	// PROCESSO 2: Sintetização/Enriquecimento (Yellowcake -> Barra de Combustível Enriquecida)
	// Nota: Como seu SynthesizeOperation atual aceita apenas um Composite de entrada na struct,
	// simulamos que o Yellowcake pré-tratado com flúor gera a barra final.
	RegisterOperation(SynthesizeOperation{
		Operation: Operation{
			ID:               "nuclear_fuel_enrichment",
			Name:             "Centrifugação e Fabricação de Célula Nuclear",
			RequiredTemp:     60.0,
			ActivationTemp:   20.0,
			Duration:         45.0,  // Processo muito lento (gargalo de automação)
			EnergyCost:       15000, // Altíssimo consumo elétrico (como centrífugas reais)
			BaseEnergyChange: 0,
			Dissipation:      0.2,
		},
		Input: []materials.Composite{
			{Substance: "uranium_dioxide", Percentual: 90},
			{Substance: "pure_iron", Percentual: 10},
		}, // Entrada: Yellowcake puro
		Output: "leu_fuel_rod", // Retorna o material completo da Célula de energia
	})

	// PROCESSO 3: A Fissão Nuclear (O que acontece dentro do Reator)
	// Consome a barra enriquecida, gera calor colossal e cospe a barra exaurida (lixo) no inventário.
	RegisterOperation(RefineOperation{
		Operation: Operation{
			ID:               "nuclear_fission",
			Name:             "Fissão Nuclear Controlada",
			RequiredTemp:     900.0,  // Temperatura operacional ideal do reator
			ActivationTemp:   200.0,  // Temperatura mínima de reação em cadeia autossustentável
			Duration:         0.0,    // Uma única barra dura MUITO tempo gerando energia (5 minutos)
			EnergyCost:       0,      // Não consome eletricidade, produz!
			BaseEnergyChange: 4500.0, // Geração de calor massiva (Muitos Joules por tick para a máquina)
			Dissipation:      15.5,   // Dissipa calor agressivamente para os blocos adjacentes
		},
		Input: "leu_fuel_rod",
		Output: []materials.Composite{
			{Substance: "cesium_137", Percentual: 70},
			{Substance: "plutonium_oxide", Percentual: 30},
		},
	})
	RegisterMachine(Machine{
		ID:         "entropy",
		Name:       "Entropia",
		Heat:       21.0,
		Procedures: []Procedure{},
	})
	RegisterMachine(Machine{
		ID:   "coal_burner",
		Name: "Fogueira de Acampamento",
		Heat: 310.0, // Iniciada com um fósforo (acima dos 300°C de ativação)
		Procedures: []Procedure{
			RefineOperations["coal_combustion"], // A fogueira está configurada para queimar carvão
		},
		EnergyBuffer: nil,
	})
	RegisterMachine(Machine{
		ID:   "crude_oil_burner",
		Name: "Tambor de Óleo",
		Heat: 310.0, // Iniciada com um fósforo (abaixo dos 400°C de ativação)
		Procedures: []Procedure{
			RefineOperations["crude_oil_combustion"], // A fogueira está configurada para queimar óleo crú
		},
		EnergyBuffer: nil,
	})
	RegisterMachine(Machine{
		ID:   "iron_bar_forger",
		Name: "Prensa Manual de Barras",
		Heat: 25.0, // Temperatura ambiente
		Procedures: []Procedure{
			// Operação que custa 500 Joules, mas gera calor interno por atrito (BaseEnergyChange = 5)
			TransformOperations["manual_press_bar"],
		},
		EnergyBuffer: energy.NewBattery(2000), // Consegue guardar até 4 ticks de buffer de energia
		TypeHint:     "electric",
	})

	RegisterMachine(Machine{
		ID:   "uranium_centrifuge",
		Name: "Centrífuga de Enriquecimento de Uranium",
		Heat: 21.0,
		Procedures: []Procedure{
			SynthesizeOperations["nuclear_fuel_enrichment"],
		},
		EnergyBuffer: energy.NewBattery(1000000),
		TypeHint:     "electric",
	})

	RegisterMachine(Machine{
		ID:   "nuclear_reactor",
		Name: "Reator Nuclear de Fissão RBMK-1000",
		Heat: 21.0, // Começa em temperatura ambiente
		Procedures: []Procedure{
			// O reator executa a operação de fissão nuclear se houver combustível no Supply
			RefineOperations["nuclear_fission"],
		},
		EnergyBuffer: energy.NewBattery(1000000), // Outras máquinas (Turbinas a Vapor) vão sugar o calor dele.
		TypeHint:     "heat",                     // O output/natureza principal desta máquina é térmico
	})
}

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

// Funções para EnergyNode e EnergyManagement ---------------------------------------------------->

func (m *Machine) Capacity() energy.EnergyUnit { return m.EnergyBuffer.Capacity() }

func (m *Machine) Stored() energy.EnergyUnit { return m.EnergyBuffer.Stored() }

// <-----------------------------------------------------------------------------------------------

// Funções de manufatura
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
	return &m // Fallback seguro
}
