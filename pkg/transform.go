package crafting

import (
	"math"

	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

// Transformar é basicamente mudar a forma de um material em outra forma

type TransformOperation struct {
	ID OperationID
	Operation
	Input  materials.MaterialID // Aqui entra o produto necessário num formato
	Output materials.MaterialID // Aqui sai o produto em outro formato
	Amount float32              // Quantidade processada por ciclo completo (Padrão: 1.0)
}

func (s TransformOperation) GetOperation() Operation { return s.Operation }
func (s TransformOperation) Kind() string            { return "transform" }

func (t TransformOperation) Execute(m *Machine, dt float32) {
	m.finishTransformation(t, dt)
}
func (m *Machine) finishTransformation(t TransformOperation, dt float32) {
	// 1. Garantir que a operação possui uma quantidade válida
	targetAmount := t.Amount
	if targetAmount <= 0 {
		targetAmount = 1.0
	}

	// 2. Verificar se o inventário contém o material de entrada necessário
	inputID := container.ItemID(t.Input)
	available := m.Supply.Materials.Items[inputID]
	if available < targetAmount {
		// Falta de matéria-prima: reseta o progresso para a máquina não "trabalhar no vácuo"
		m.Progress = 0
		return
	}

	// 3. Gerenciamento do Tempo e Progresso da Operação
	// Se a duração for 0, o processo é instantâneo por tick
	if t.Operation.Duration <= 0 {
		m.executeTransformationSlice(t, targetAmount)
		return
	}

	// Acumula o progresso com base no delta time útil
	m.Progress += dt

	// Se o progresso atingiu ou superou a duração necessária, finaliza o lote/item
	if m.Progress >= t.Operation.Duration {
		// Calcula quantos ciclos completos cabem neste estouro de tempo (útil para dt alto)
		cycles := float32(math.Floor(float64(m.Progress / t.Operation.Duration)))
		totalToProcess := targetAmount * cycles

		// Valida se há estoque para múltiplos ciclos acumulados
		if available < totalToProcess {
			totalToProcess = float32(math.Floor(float64(available/targetAmount))) * targetAmount
		}

		if totalToProcess > 0 {
			m.executeTransformationSlice(t, totalToProcess)
		}

		// Mantém o resto do frame para o próximo tick (evita perda de precisão temporal)
		m.Progress = float32(math.Mod(float64(m.Progress), float64(t.Operation.Duration)))
	}
}

// executeTransformationSlice faz a troca física dos itens nos contêineres de forma atômica
func (m *Machine) executeTransformationSlice(t TransformOperation, amount float32) {
	// Remove do inventário de entrada
	removed, ok := m.Supply.Materials.RemoveItem(container.ItemID(t.Input), amount)
	if !ok || removed <= 0 {
		return
	}

	// Busca a definição original do material para manter a coerência das propriedades físicas
	origMaterial, err := materials.GetMaterial(t.Output)
	if err != nil {
		// Se o material de saída não existir no registro, devolve o input para não sumir com os itens
		m.Supply.Materials.AddItem(container.ItemID(t.Input), removed)
		return
	}

	// Modifica dinamicamente as propriedades da instância do material gerado
	// O material transformado assume a temperatura interna atual da câmara da máquina
	transformedInstance := origMaterial
	transformedInstance.Temperature = m.Heat

	// Adiciona o material transformado ao inventário de saída da máquina
	// Se o seu sistema de Storage aceitar structs complexas, usamos a instância;
	// Caso use IDs, passamos o ID e a quantidade.
	m.Supply.Materials.AddItem(container.ItemID(transformedInstance.ID), removed)
}
