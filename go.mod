module github.com/GoGamesLab/Crafting

go 1.27

require (
	github.com/GoGamesLab/Energy v0.0.0
	github.com/GoGamesLab/Inventory v0.0.0
	github.com/GoGamesLab/Materials v0.0.0
)

replace (
	github.com/GoGamesLab/Energy => ../Energy
	github.com/GoGamesLab/Inventory => ../Inventory
	github.com/GoGamesLab/Materials => ../Materials
)
