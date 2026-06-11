package crafting

import (
	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

// Basicamente sintetizar é processar uma série de elementos e torná-los em materiais possíveis

type SynthesizeOperation struct {
	Operation
	Input  []materials.Composite // Aqui entram as substâncias necessárias e ficam os resíduos/excedentes (Escória/Gases)
	Output materials.MaterialID  // Aqui é informado o material desejado
}

func (s SynthesizeOperation) GetOperation() Operation { return s.Operation }
func (s SynthesizeOperation) Kind() string            { return "synthesize" }

func (s SynthesizeOperation) Execute(m *Machine, dt float32) {
	m.finishSynthesization(s, dt)
}

func (m *Machine) finishSynthesization(s SynthesizeOperation, dt float32) {
	// Lógica:
	// Obtém lista de materiais possíveis com os composites informados
	possibleMats, err := materials.Compositions(s.Input)
	if err != nil {
		return
	}
	for _, possible := range possibleMats {
		if s.Output == possible.ID {
			// 1. Consome os itens (Composites) do estoque de entrada
			// na proporção da composição do produto desejado (s.Output)
			// 2. Cria o novo Material (ex: Barra de Aço)
			material, err := materials.GetMaterial(s.Output)
			if err != nil {
				return
			}
			elements := material.Reduction(1) // calcula os elementos para sintetizar uma unidade
			for id, amount := range elements {
				available := m.Supply.Substances.Items[container.ItemID(id)]
				if amount > available {
					return // não tem substância suficiente
				} else {
					Consume(m, id, amount) // consome o necessário da substância
				}
			}
			Produce(m, material.ID, 1) // produz a unidade após consumir todas substâncias necessárias

			break // terminou de produzir o material desejado
		}
	}
}
