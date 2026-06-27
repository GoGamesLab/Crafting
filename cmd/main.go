package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	crafting "github.com/GoGamesLab/Crafting/pkg"
	energy "github.com/GoGamesLab/Energy/pkg"
	container "github.com/GoGamesLab/Inventory/pkg"
	materials "github.com/GoGamesLab/Materials/pkg"
)

var Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

func burnFuel(machineID crafting.MachineID, fuelID materials.MaterialID, amount float32) crafting.Supply {
	machine, err := crafting.GetNewMachine(machineID)
	if err != nil {
		log.Fatalf("💣 error: %s", err)
	}
	machine.SetupEnergyManager()
	machine.SetupSupply()
	machine.EnergyManager.RegisterSource("fogueira", machine)
	machine.EnergyManager.RegisterSink("fogueira", machine)
	material, _ := materials.GetMaterial(fuelID)

	Logger.Info(fmt.Sprintf("Iniciando combustão: %f unidades de %s na máquina %s", amount, material.Name, machine.Name))

	crafting.Store(machine.Supply, fuelID, amount)
	for machine.GetSupply().Materials.Items[container.ItemID(fuelID)] > 0 {
		machine.Update(1)
	}

	Logger.Info(fmt.Sprintf("⚡️ Produzido %f EUs", machine.EnergyManager.Produced))
	Logger.Info(fmt.Sprintf("⚡️ Consumido %f EUs", machine.EnergyManager.Consumed))
	Logger.Info(fmt.Sprintf("⚡️ Energia residual %f EUs", machine.EnergyManager.GetEnergy()))

	return machine.GetSupply()
}

func buildReactor() *crafting.Machine {
	reactor, err := crafting.GetNewMachine("nuclear_reactor")
	if err != nil {
		log.Fatalf("💣 error: %s", err)
	}
	reactor.SetupEnergyManager()
	reactor.SetupSupply()
	reactor.EnergyManager.RegisterSource("reactor-1", reactor)

	centrifuge, err := crafting.GetNewMachine("uranium_centrifuge")
	if err != nil {
		log.Fatalf("💣 error: %s", err)
	}
	centrifuge.SetEnergyManager(reactor.EnergyManager)
	centrifuge.SetSupply(reactor.Supply)
	reactor.EnergyManager.RegisterSink("centrifuge-1", centrifuge)
	crafting.Store(centrifuge.Supply, materials.SubstanceID("uranium_dioxide"), 0.45)
	crafting.Store(centrifuge.Supply, materials.SubstanceID("pure_iron"), 0.005)
	for true {
		centrifuge.Update(1)
		if centrifuge.Supply.Materials.Items[container.ItemID("leu_fuel_rod")] >= 0.0001 {
			break
		}
	}

	reactor.EnergyManager.RegisterSource("reactor-1", reactor)

	printResidues(*reactor.Supply, "Estoque")

	return reactor
}

func buildPrensa(em *energy.EnergyManager, s *crafting.Supply) *crafting.Machine {
	prensa, err := crafting.GetNewMachine("iron_plate_forger")
	if err != nil {
		panic(err)
	}
	prensa.SetEnergyManager(em)
	prensa.SetSupply(s)

	em.RegisterSink("prensa-1", prensa)

	return prensa
}

func factory() (crafting.Supply, crafting.Supply) {
	reactor := buildReactor()

	f := crafting.GetNewFactory()
	crafting.Store(f.Warehouse, materials.MaterialID("iron_bar"), 100)
	crafting.Store(f.Warehouse, materials.MaterialID("iron_plate"), 0)

	const machines = 1000
	for range machines {
		f.Lines = append(f.Lines, crafting.Line{})
		f.Lines[0].Machines = append(f.Lines[0].Machines, buildPrensa(reactor.EnergyManager, f.Warehouse))
	}

	printResidues(*f.Warehouse, "Estoque")

	dt := float32(1.0)
	for true {
		if reactor.Supply.Materials.Items[container.ItemID("leu_fuel_rod")] == 0 {
			break
		}

		reactor.Update(dt)
		f.Update(dt)
	}

	f.TransferToWarehouse()

	Logger.Info(fmt.Sprintf("⚡️ Produzido %f EUs", reactor.EnergyManager.Produced))
	Logger.Info(fmt.Sprintf("⚡️ Consumido %f EUs", reactor.EnergyManager.Consumed))
	Logger.Info(fmt.Sprintf("⚡️ Energia residual %f EUs", reactor.EnergyManager.GetEnergy()))

	return reactor.GetSupply(), *f.Warehouse
}

func printResidues(i crafting.Supply, t string) {
	for id, j := range i.Materials.Items {
		m, _ := materials.GetMaterial(materials.MaterialID(id))
		Logger.Info(fmt.Sprintf("📦 %s %f material %s", t, j, m.Name))
	}
	for id, j := range i.Substances.Items {
		s, _ := materials.GetSubstance(materials.SubstanceID(id))
		Logger.Info(fmt.Sprintf("🧪 %s %f substância %s", t, j, s.Name))
	}
	for id, j := range i.Elements.Items {
		e, _ := materials.GetElement(materials.ElementID(id))
		Logger.Info(fmt.Sprintf("⚛️️ %s %f elemento %s", t, j, e.Name))
	}
}

func main() {
	materials.LoadElementsFromJSON("../../Grind/assets/resources/elements.json")
	materials.LoadSubstancesFromJSON("../../Grind/assets/resources/substances.json")
	materials.LoadMaterialsFromJSON("../../Grind/assets/resources/materials.json")

	crafting.LoadDefinitionsFromJSON("../../Grind/assets/craft/definitions.json")

	energy.LoadConvertersFromJSON("../../Grind/assets/energy/converters.json")

	reactor, production := factory()
	printResidues(production, "Produzido")
	printResidues(reactor, "Produzido")

	printResidues(burnFuel("coal_burner", "coal", 10), "Produzido")
	printResidues(burnFuel("crude_oil_burner", "crude_oil", 10), "Produzido")
}
