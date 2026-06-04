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

	crafting.Store(&machine, fuelID, amount)
	for machine.GetSupply().Materials.Items[container.ItemID(fuelID)] > 0 {
		machine.Update(1)
		Logger.Info(fmt.Sprintf("🔥 [%s] Temperatura: %.2f°C", machine.Name, machine.Heat))
	}
	return machine.GetSupply()
}

func prensaMetais() crafting.Supply {
	energyManager := energy.NewEnergyManager()
	energyManager.RegisterConverter(energy.NewElectricConverter(0.85))

	// 1. Instancia o Reator Nuclear
	reactor := energy.NewReactor(0.75, 1000000, 10.0)
	reactor.AddFuel(1000) // Adiciona combustível bruto
	energyManager.RegisterNode("reactor-1", reactor)

	prensarMetaisOpID := crafting.OperationID(len(crafting.TransformOperations))
	crafting.RegisterOperation(crafting.TransformOperation{
		Operation: crafting.Operation{
			ID:               prensarMetaisOpID,
			Name:             "Prensa metais",
			ActivationTemp:   20.0,
			Duration:         15.0,
			EnergyCost:       500,
			BaseEnergyChange: 5.0,
			Dissipation:      0.1,
		},
		Input:  materials.MaterialID(materials.PureIronID),
		Output: materials.MaterialID(materials.PureIronID),
	})

	// 2. Instancia uma Prensa Hidráulica/Elétrica (Consome 500 Joules/s, precisa de 50°C para operar)
	prensaHidraulicaEletricaID := crafting.MachineID(len(crafting.Machines))
	crafting.RegisterMachine(crafting.Machine{
		ID:   prensaHidraulicaEletricaID,
		Name: "Prensa Hidráulica Elétrica",
		Heat: 25.0, // Temperatura ambiente
		Procedures: []crafting.Procedure{
			// Operação fictícia que custa 500 Joules, mas gera calor interno por atrito (BaseEnergyChange = 5)
			crafting.TransformOperations[prensarMetaisOpID],
		},
		EnergyManager: energyManager,
		EnergyBuffer:  energy.NewBattery(2000), // Consegue guardar até 4 ticks de buffer de energia
		TypeHint:      "electric",
	})

	prensa := crafting.GetMachine(prensaHidraulicaEletricaID)

	// 3. O Cabo que liga o Reator à Prensa
	cabo := &energy.EnergyPipe{ThroughputPerTick: 1000, LossPerUnitDistance: 0.01, Distance: 5}

	// --- Loop do Simulador (Simulando 5 Ticks) ---
	dt := float32(1.0)
	for i := range 5 {
		// A) Reator queima combustível internamente e gera energia para o seu próprio buffer
		reactor.Update()

		// B) O cabo puxa energia do buffer do Reator e tenta empurrar para o buffer da Prensa
		demand := prensa.Demand() * energy.EnergyUnit(dt)
		transferred, pipeByproducts, _ := cabo.Transfer(reactor, &prensa, demand)
		for _, b := range pipeByproducts {
			Logger.Info(fmt.Sprintf("- byproduct %s = %f", b.Type, b.Value))
		}

		// C) Executa o Update da máquina
		prensa.Update(dt)

		fmt.Printf("Tick %d -> Cabo transmitiu: %.2f J | Buffer Prensa: %.2f J | Temp Prensa: %.2f°C\n",
			i, transferred, prensa.EnergyBuffer.Stored(), prensa.Heat)
	}

	return prensa.GetSupply()
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
	printResidues(burnFuel(crafting.CoalBurnerID, materials.CoalID, 10))

	printResidues(burnFuel(crafting.CrudeOilBurnerID, materials.CrudeOilID, 10))

	printResidues(prensaMetais())
}
