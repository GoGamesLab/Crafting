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
	material, _ := materials.GetMaterial(fuelID)

	Logger.Info(fmt.Sprintf("Iniciando combustão: %f unidades de %s na máquina %s", amount, material.Name, machine.Name))

	crafting.Store(machine, fuelID, amount)
	for machine.GetSupply().Materials.Items[container.ItemID(fuelID)] > 0 {
		machine.Update(1)
		//Logger.Info(fmt.Sprintf("🔥 [%s] Temperatura: %.2f°C", machine.Name, machine.Heat))
	}
	return machine.GetSupply()
}

func buildReactor() *crafting.Machine {
	reactor, err := crafting.GetNewMachine("nuclear_reactor")
	if err != nil {
		log.Fatalf("💣 error: %s", err)
	}
	reactor.SetupEnergyManager()
	reactor.EnergyManager.RegisterSource("reactor-1", reactor)

	fuel, _ := materials.GetMaterial("leu_fuel_rod")
	crafting.Store(reactor, fuel.ID, 1)

	centrifuge, err := crafting.GetNewMachine("uranium_centrifuge")
	if err != nil {
		log.Fatalf("💣 error: %s", err)
	}
	centrifuge.SetEnergyManager(reactor.EnergyManager)
	reactor.EnergyManager.RegisterSink("centrifuge-1", centrifuge)
	crafting.Store(centrifuge, materials.SubstanceID("uranium_dioxide"), 9000)
	crafting.Store(centrifuge, materials.SubstanceID("pure_iron"), 100)
	for true {
		centrifuge.Update(1)
		if centrifuge.Supply.Materials.Items[container.ItemID("leu_fuel_rod")] >= 10 {
			break
		}
	}

	crafting.Store(reactor, fuel.ID, centrifuge.Supply.Materials.Items[container.ItemID(fuel.ID)])

	reactor.EnergyManager.RegisterSource("reactor-1", reactor)

	crafting.Store(reactor, fuel.ID, centrifuge.Supply.Materials.Items[container.ItemID(fuel.ID)])

	return reactor
}

func buildPrensa(em *energy.EnergyManager) crafting.Machine {
	prensa, err := crafting.GetNewMachine("iron_plate_forger")
	if err != nil {
		panic(err)
	}
	prensa.SetEnergyManager(em)
	crafting.Store(prensa, materials.MaterialID("iron_bar"), 100)
	crafting.Store(prensa, materials.MaterialID("iron_plate"), 0)

	em.RegisterSink("prensa-1", prensa)

	return *prensa
}

func factory() (crafting.Supply, crafting.Supply) {
	f := crafting.GetNewFactory()

	const machines = 1
	reactor := buildReactor()
	for range machines {
		f.Machines = append(f.Machines, buildPrensa(reactor.EnergyManager))
	}

	dt := float32(1.0)
	for range 90 {
		reactor.Update(dt)
		f.Update(dt)
	}

	f.TransferToWarehouse()

	return reactor.GetSupply(), *f.Warehouse
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

	reactor, production := factory()
	printResidues(production)
	printResidues(reactor)

	printResidues(burnFuel("coal_burner", "coal", 10))
	printResidues(burnFuel("crude_oil_burner", "crude_oil", 10))
}
