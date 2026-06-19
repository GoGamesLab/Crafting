package crafting

import (
	"encoding/json"
	"errors"
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
	ID             MachineID `json:"ID"`
	Name           string    `json:"Name"`
	Heat           float32   `json:"Heat"`
	ProcedureIDs   []string  `json:"ProcedureIDs"`
	EnergyCapacity float64   `json:"EnergyCapacity"` // Usado para criar o EnergyBuffer
	TypeHint       string    `json:"TypeHint"`
}

type OperationID string
type MachineID string

type Operation struct {
	ID               OperationID
	Name             string
	RequiredTemp     float32           // Condição térmica para manter a reação
	EnergyCost       energy.EnergyUnit // Energia (Joules) necessária para a reação
	Duration         float32           // Tempo em segundos
	BaseEnergyChange float32           // Negativo = Consome | Positivo = Produz
	ActivationTemp   float32           // Temperatura mínima para a reação começar
	Dissipation      float32           // Índice de dissipação
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
	Heat          float32 // Temperatura interna atual
	Procedures    []Procedure
	Progress      float32
	Supply        *Supply
	EnergyBuffer  *energy.Battery       // Buffer interno de Joules recebidos da rede externa (opcional)
	TypeHint      string                // "electric", "kinetic", "heat" -- o que o motor dela aceita
	EnergyManager *energy.EnergyManager // conversores
}

func energyRegime(m *Machine, p Operation, dt float32) (hasEnergy bool) {
	hasEnergy = true
	if p.EnergyCost > 0 {
		needed := p.EnergyCost * energy.EnergyUnit(dt)
		var consumed energy.EnergyUnit

		if m.EnergyBuffer != nil {
			// Tenta retirar do buffer interno alimentado por cabos
			consumed, _ = m.EnergyBuffer.Consume(needed)
		}

		// Se não havia energia suficiente estocada no cabo/bateria interna,
		// faz uma requisição direta ao EnergyManager em modo emergencial/contingência
		if consumed < needed && m.EnergyManager != nil {
			req := energy.EnergyRequirement{
				Amount:   needed - consumed,
				TypeHint: m.TypeHint,
			}
			// Passa o ID da própria máquina na lista de nós preferenciais, se aplicável
			provided, byproducts, err := m.EnergyManager.SatisfyRequirement(req, []string{fmt.Sprint(m.ID)})
			if err == nil {
				consumed += provided
				// Se o conversor elétrico gerou calor residual (perda do motor),
				// esse calor é transferido diretamente para a estrutura física da máquina!
				for _, bp := range byproducts {
					if bp.Type == "heat" {
						m.Heat += float32(bp.Value)
					}
				}
			}
		}

		// Se mesmo com a rede a energia foi insuficiente, a máquina engasga (não avança a operação)
		if consumed < needed {
			hasEnergy = false
			// Aqui você pode disparar um log ou evento visual de "Falta de Energia"
		}
	}

	return
}

func (m *Machine) Update(dt float32) {
	var to energy.EnergyUnit
	// executar processos do energy manager (se houver)
	// o último processo vai guardar a energia convertida na bateria, e portanto teria
	// que ser eletricidade (aqui a ordem importa)
	// então, se tiver bateria, provavelmente deveria garantir que eletricidade seria
	// o último conversor e ir lendo de trás para frente até chegar em eletricidade
	//
	// por exemplo, se eu tiver:
	// [
	//     {
	//         "id": "boiler",
	//         "from_type": "heat",
	//         "to_type": "kinetic",
	//         "efficiency": 0.85
	//     },
	//     {
	//         "id": "dynamo",
	//         "from_type": "kinetic",
	//         "to_type": "electric",
	//         "efficiency": 0.90
	//     },
	//     {
	//         "id": "solar_panel",
	//         "from_type": "solar",
	//         "to_type": "electric",
	//         "efficiency": 0.20
	//     },
	//     {
	//         "id": "eolic",
	//         "from_type": "wind",
	//         "to_type": "kinetic",
	//         "efficiency": 0.65
	//     },
	//     {
	//         "id": "hydric",
	//         "from_type": "water",
	//         "to_type": "kinetic",
	//         "efficiency": 0.75
	//     }
	// ]
	// e se eu tiver bateria (EnergyBuffer), então procurar id "electric"
	// se não tiver "electric" em to_type é erro (se tem bateria tem que ter eletricidade)
	// a máquina possui o TypeHint que é a energia que ela produz
	// {
	//     "ID": "nuclear_reactor",
	//     "Name": "Reator Nuclear de Fissão RBMK-1000",
	//     "Heat": 21.0,
	//     "ProcedureIDs": [
	//         "nuclear_fission"
	//     ],
	//     "EnergyCapacity": 1000000,
	//     "TypeHint": "heat"
	// }
	//
	// assim eu tenho que partir de calor e chegar em eletricidade usando todos os conversores
	// registrados para esta máquinas
	//
	// battery (electric) < dynamo(electric <- kinetic) < boiler(kinetic <- heat)
	if m.EnergyManager != nil {
		for _, c := range m.EnergyManager.EnergyConverters {
			to = c.FromBase(1)
			if _, err := c.ToBase(to); err != nil {
				log.Fatalf("🧨 converter: %s", err)
			}
		}
	}

	// ao final, se houver energia elétrica, transferir para a bateria interna (se houver)
	// mas como garantir que a bateria receba o tipo de energia desejado (sempre elétrica)?
	if m.EnergyBuffer != nil {
		m.EnergyBuffer.Produce(to)
	}
	// se não houver bateria interna, então basicamente o TypeHint da máquina muda para o que
	// resultar em to

	for _, step := range m.Procedures {
		p := step.GetOperation()

		// Check de Ativação
		if energyRegime(m, p, dt) && m.Heat >= p.ActivationTemp {
			// Execução
			m.executeStep(step, dt)

			m.Heat = calculateTemperature(m.Heat, p.BaseEnergyChange*dt, p.Dissipation)
		} else {
			// Balanço Térmico
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

	// 1. Carregar e Registrar Operações Primeiro
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

	// 2. Carregar e Registrar Máquinas
	for _, mJSON := range gameData.Machines {
		m := Machine{
			ID:       mJSON.ID,
			Name:     mJSON.Name,
			Heat:     mJSON.Heat,
			TypeHint: mJSON.TypeHint,
		}

		// Inicializar o EnergyBuffer se houver capacidade declarada
		if mJSON.EnergyCapacity > 0 {
			m.EnergyBuffer = energy.NewBattery(mJSON.EnergyCapacity)
		}

		// Vincular os Procedures pelas strings de ID
		for _, procID := range mJSON.ProcedureIDs {
			proc, err := findProcedure(OperationID(procID))
			if err != nil {
				return fmt.Errorf("máquina %s referenciou procedure inexistente: %w", m.ID, err)
			}
			m.Procedures = append(m.Procedures, proc)
		}

		// Registrar (Isso também inicializará o Supply)
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

func (m *Machine) Produce(amount energy.EnergyUnit) (energy.EnergyUnit, error) {
	if m.EnergyBuffer == nil {
		return 0, errors.New("machine cannot store work energy")
	}
	return m.EnergyBuffer.Produce(amount)
}

func (m *Machine) Consume(amount energy.EnergyUnit) (energy.EnergyUnit, error) {
	if m.EnergyBuffer == nil {
		return 0, nil
	}
	return m.EnergyBuffer.Consume(amount)
}

func (m *Machine) Peek() energy.EnergyUnit {
	if m.EnergyBuffer == nil {
		return 0
	}
	return m.EnergyBuffer.Peek()
}

func (m *Machine) Demand() energy.EnergyUnit {
	if len(m.Procedures) == 0 {
		return 0
	}
	// A demanda instantânea para este tick base/segundo é o custo da operação atual
	return m.Procedures[0].GetOperation().EnergyCost
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

	// 1. Acumula o progresso baseado no tempo real que passou
	m.Progress += dt

	// 2. Verifica se o tempo de duração foi atingido
	// isso aqui não está sendo um processo contínuo
	// mas tem um desempenho melhor (só calcular tudo ao final do tempo)
	if m.Progress >= p.Duration {
		// O PROCESSO TERMINOU: O objeto Procedure agora decide como se executar
		step.Execute(m, dt)

		// 3. Reseta o progresso para o próximo ciclo (ou próximo passo da chain)
		m.Progress = 0
	}
}

// calculateTemperature calcula a temperatura no próximo tick.
// current: temperatura atual
// heatGain: ganho de temperatura por tick enquanto acesa
// retorna: nova temperatura após um tick
func calculateTemperature(current, heatGain float32, dissipation float32) float32 {
	const Tenv = 21.0

	// Tempo de meia-vida em ticks para a diferença T - Tenv.
	// Ajuste aqui para alterar quão rápido a fornalha dissipa calor.
	// Ex.: dissipation = 50 → dissipa mais devagar; dissipation = 10 → mais rápido.

	// taxa de dissipação por tick derivada da meia-vida:
	// fator de retenção por tick = 2^(-1/tHalf)
	retain := math.Pow(2.0, -1.0/float64(dissipation))
	k := 1.0 - retain // fração perdida por tick

	// Atualiza: acumula ganho e aplica dissipação em relação ao ambiente
	delta := float64(current - Tenv)
	delta = delta*(1.0-k) + float64(heatGain) // primeiro aplica retenção, depois adiciona ganho
	// equivalência ao modelo T_next = Tenv + (T_current - Tenv)*(1-k) + heatGain

	return float32(Tenv + delta)
}

// findProcedure busca a operação pelo ID em todos os registros conhecidos.
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
