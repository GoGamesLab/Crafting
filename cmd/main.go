package main

import (
	"fmt"
	"log/slog"
	"os"

	crafting "github.com/GoGamesLab/Crafting/pkg"
	energy "github.com/GoGamesLab/Energy/pkg"
	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

var Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

func burnFuel(machineID crafting.MachineID, fuelID materials.MaterialID, amount float32) crafting.Supply {
	machine := crafting.GetMachine(machineID)
	material, _ := materials.GetMaterial(fuelID)

	Logger.Info(fmt.Sprintf("Iniciando combustão: %f unidades de %s na máquina %s", amount, material.Name, machine.Name))

	crafting.Store(machine, fuelID, amount)
	for machine.GetSupply().Materials.Items[container.ItemID(fuelID)] > 0 {
		machine.Update(1)
		Logger.Info(fmt.Sprintf("🔥 [%s] Temperatura: %.2f°C", machine.Name, machine.Heat))
	}
	return machine.GetSupply()
}

func prensaMetais() (crafting.Supply, crafting.Supply) {
	// 1. Instancia o Reator Nuclear (vem com energy manager acoplado)
	//reactor := energy.NewUraniumReactor(0.75, 1000000, 10.0)
	reactor := crafting.GetMachine("nuclear_reactor")

	dynamo, err := energy.GetConverter("dynamo")
	if err != nil {
		Logger.Error(fmt.Sprintf("💣 conversor error: %s", err))
	}
	reactor.EnergyManager.RegisterConverter(dynamo) // kinetic -> electric

	boiler, err := energy.GetConverter("boiler")
	if err != nil {
		Logger.Error(fmt.Sprintf("💣 conversor error: %s", err)) // heat -> kinetic
	}
	reactor.EnergyManager.RegisterConverter(boiler) // -> heat

	// TODO: ler o que fiz em Crafting/pkg/crafting.go onde a função Machine.Update() vai
	// processar a conversão de energia do reator (como ele tem conversores de energia
	// registrados eles serão executados antes da execução dos passos)
	// -- como no início o reator não tem calor algum, não vai haver conversão
	// mas assim que esquentar o reator produz eletricidade que vai parar na bateria
	// CADA VEZ MAIS PENSO EM USAR PROCESSOS DE MÁQUINAS PARA PRODUÇÃO E CONVERSÃO DE ENERGIA
	// COMO FAZEMOS COM SÍNTESE E REFINO

	// produz um mínimo inicial com combustível "importado"
	fuel, _ := materials.GetMaterial("leu_fuel_rod")
	crafting.Store(reactor, fuel.ID, 1)
	reactor.Update(1) // para esquentar
	reactor.Update(1) // para produzir algo
	// mas é claro que não está produzindo, pois nada no sistema de manufatura "produz" energia
	// energia não é um produto de manufatura
	// o reator só produz calor, então eu preciso aplicar:
	// - um transformador de calor em energia cinética, seguido de
	// - um transformador de energia cinética em eletricidade
	// agora eu tenho os conversores de energia e posso ligá-los ao reator usando o energy manager:
	// {
	//     "id": "boiler",
	//     "from_type": "heat",
	//     "to_type": "kinetic",
	//     "efficiency": 0.85
	// },
	// {
	//     "id": "dynamo",
	//     "from_type": "kinetic",
	//     "to_type": "electric",
	//     "efficiency": 0.90
	// },
	// MAS ISSO ESTÁ ERRADO! os conversores deveriam estar acoplados ao reator como processos
	// de produção de energia!
	// isso implica em: ou Machine é "ciente" de Energy (já é, pois usa Battery)!
	// ou a bateria teria os conversores acoplados, assim todo o processo de conversão ocorre
	// no pacote Energy durante o Update!
	// e aí vem o meu grande erro! o energy manager é um gerenciador por fonte de energia
	// e não um singleton de todo o sistema (até poderia ser), e por isso os conversores estão no EM

	// e somente então armazenar a eletricidade na bateria interna do reator!

	// produzir combustível para o reator
	// a centrífuga não está funcionando por falta de energia!
	// será que consigo usar o reator?
	// agora que criei a centrífuga para produzir o combustível do reator, ela tem o próprio energy manager
	// o que eu faço? a centrífuga não ter energia e o cabo não faz nada!
	centrifuge := crafting.GetMachine("uranium_centrifuge")
	reactor.EnergyManager.RegisterNode("centrifuge-1", centrifuge)
	cabo1 := &energy.EnergyPipe{ThroughputPerTick: 1000, LossPerUnitDistance: 0.01, Distance: 5}
	// tentei transferir do reator para a centrífuga, mas o reator ainda não tem energia!
	cabo1.Transfer(reactor, centrifuge, 200000)
	crafting.Store(centrifuge, materials.SubstanceID("uranium_dioxide"), 9000)
	crafting.Store(centrifuge, materials.SubstanceID("pure_iron"), 100)
	for true {
		centrifuge.Update(1)
		if centrifuge.Supply.Materials.Items[container.ItemID("leu_fuel_rod")] >= 10 {
			break
		}
	}

	//reactor.AddFuel(1000) // Adiciona combustível bruto

	// ainda não sei bem para que servem esses nodes
	// e como mais tarde vou processor a transferência de energia do
	// reator para o boiler e deste para o dinamo
	// esse registro agora não faz sentido já que o EM é do próprio reator!
	reactor.EnergyManager.RegisterNode("reactor-1", reactor)

	prensarMetaisOpID := crafting.OperationID("prensa_metais")
	crafting.RegisterOperation(crafting.TransformOperation{
		Operation: crafting.Operation{
			ID:               prensarMetaisOpID,
			Name:             "Prensa metais",
			ActivationTemp:   20.0,
			Duration:         2.0,
			EnergyCost:       500,
			BaseEnergyChange: 5.0,
			Dissipation:      0.1,
		},
		Input:  "iron_bar",
		Output: "iron_plate",
		Amount: 1,
	})

	// 2. Instancia uma Prensa Hidráulica/Elétrica (Consome 500 Joules/s, precisa de 50°C para operar)
	prensaHidraulicaEletricaID := crafting.MachineID("prensa_hidraulica")
	crafting.RegisterMachine(crafting.Machine{
		ID:   prensaHidraulicaEletricaID,
		Name: "Prensa Hidráulica Elétrica",
		Heat: 25.0, // Temperatura ambiente
		Procedures: []crafting.Procedure{
			// Operação fictícia que custa 500 Joules, mas gera calor interno por atrito (BaseEnergyChange = 5)
			crafting.TransformOperations[prensarMetaisOpID],
		},
		EnergyBuffer: energy.NewBattery(2000), // Consegue guardar até 4 ticks de buffer de energia
		TypeHint:     "electric",
	})

	prensa := crafting.GetMachine(prensaHidraulicaEletricaID)
	crafting.Store(prensa, materials.MaterialID("iron_bar"), 100)
	crafting.Store(prensa, materials.MaterialID("iron_plate"), 0)

	// isso significa que tenho que colocar toda a rede elétrica registrada
	// no reator, ao invés de ter um sistema de distribuição (usando os cabos)
	reactor.EnergyManager.RegisterNode("prensa-1", prensa)

	// 3. O Cabo que liga o Reator à Prensa
	cabo2 := &energy.EnergyPipe{ThroughputPerTick: 1000, LossPerUnitDistance: 0.01, Distance: 5}

	// --- Loop do Simulador (Simulando 5 Ticks) ---
	dt := float32(1.0)
	for i := range 15 {
		// A) Reator queima combustível internamente e gera energia para o seu próprio buffer
		reactor.Update(1)

		// B) O cabo puxa energia do buffer do Reator e tenta empurrar para o buffer da Prensa
		demand := prensa.Demand() * energy.EnergyUnit(dt)
		transferred, pipeByproducts, _ := cabo2.Transfer(reactor, prensa, demand)
		for _, b := range pipeByproducts {
			Logger.Info(fmt.Sprintf("- byproduct %s = %f", b.Type, b.Value))
		}

		// C) Executa o Update da máquina
		prensa.Update(dt)

		fmt.Printf("Tick %d -> Cabo transmitiu: %.2f J | Buffer Prensa: %.2f J | Temp Prensa: %.2f°C | Buffer Reator: %.2f J | Temp Reator: %.2f°C \n",
			i, transferred, prensa.EnergyBuffer.Stored(), prensa.Heat, reactor.EnergyBuffer.Stored(), reactor.Heat)
	}

	return prensa.GetSupply(), reactor.GetSupply()
}

func printResidues(i crafting.Supply) {
	for id, j := range i.Materials.Items {
		m, _ := materials.GetMaterial(materials.MaterialID(id))
		Logger.Info(fmt.Sprintf("📦 Produzido %f material %s", j, m.Name))
	}
	for id, j := range i.Substances.Items {
		s, _ := materials.GetSubstance(materials.SubstanceID(id))
		Logger.Info(fmt.Sprintf("🧪 Produzido %f substância %s", j, s.Name))
	}
	for id, j := range i.Elements.Items {
		e, _ := materials.GetElement(materials.ElementID(id))
		Logger.Info(fmt.Sprintf("⚛️️ Produzido %f elemento %s", j, e.Name))
	}
}

func main() {
	materials.LoadElementsFromJSON("../../Grind/assets/resources/elements.json")
	materials.LoadSubstancesFromJSON("../../Grind/assets/resources/substances.json")
	materials.LoadMaterialsFromJSON("../../Grind/assets/resources/materials.json")

	crafting.LoadDefinitionsFromJSON("../../Grind/assets/craft/definitions.json")

	energy.LoadConvertersFromJSON("../../Grind/assets/energy/converters.json")

	//printResidues(burnFuel("coal_burner", "coal", 10))

	//printResidues(burnFuel("crude_oil_burner", "crude_oil", 10))

	machine, reactor := prensaMetais()
	printResidues(machine)
	printResidues(reactor)
}
