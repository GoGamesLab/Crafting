package main

import (
	"fmt"
	"log/slog"
	"os"

	crafting "github.com/GoGamesLab/Crafting/pkg"
	"github.com/GoGamesLab/Inventory/pkg/container"
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
}
