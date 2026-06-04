package crafting

import (
	"errors"
	"fmt"
	"math"

	energy "github.com/GoGamesLab/Energy/pkg"
	"github.com/GoGamesLab/Inventory/pkg/container"
	materials "github.com/GoGamesLab/Materials/pkg"
)

type OperationID uint16
type ProcedureID uint16
type MachineID uint16
type ProductID uint16

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

type MachineType int

const (
	Manual MachineType = iota
	Mixer
	Synthesizer
	Processor
	Extruder
	Laminator
	Refiner
	Assembler
	Smelter
)

type Procedure interface {
	GetOperation() Operation
	Kind() string
	Execute(m *Machine, dt float32)
}

type ProductType int

const (
	MaterialType ProductType = iota
	SubstanceType
)

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
	EnergyManager *energy.EnergyManager
	EnergyBuffer  *energy.Battery // Buffer interno de Joules recebidos da rede externa (opcional)
	TypeHint      string          // "electric", "kinetic", "heat" -- o que o motor dela aceita
}

func (m *Machine) Update(dt float32) {
	for _, step := range m.Procedures {
		p := step.GetOperation()

		// Energy regime
		hasEnergy := true
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
				continue
			}
		}

		if hasEnergy {
			// Check de Ativação
			if m.Heat >= p.ActivationTemp {
				// Execução
				m.executeStep(step, dt)

				m.Heat = calculateTemperature(m.Heat, p.BaseEnergyChange*dt, p.Dissipation)
			} else {
				// Balanço Térmico
				m.Heat += p.BaseEnergyChange * dt
			}
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
