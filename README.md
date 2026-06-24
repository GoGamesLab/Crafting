# TODO

## Produção de Energia

A produção de energia deveria passar pelo processo de sintetização, que consome material e gera resíduos, mas atualmente está apenas executando `EnergyManager.EnergySources[].GenerateEnergy(amount)`, sem consumir material algum.

EnergySource é uma interface para Machine, que portanto poderia de algum modo permitir a invocação do Execute de uma operação que poderia ser Refine, Synthesize ou Transform, dependendo da fonte produtora. Uma fogueira ou um reator nuclear usariam Refine para transoformar o combustível em calor e alimentar uma caldeira que por transformação alimentaria uma turbina e produziria eletricidade por exemplo.

Isso irá permitir que as fontes produtoras de energia consumam materiais como qualquer máquina, e a máquina deverá também consumir energia em seu processo, o que implica que a máquina deverá também ser composta de conversores para por exemplo transformar eletricidade em movimento que irá por exemplo executar a transformação, ou eletricidade em calor que irá executar a síntese ou refino!

## Transferência de materiais na linha de produção

Uma simplificação bem vinda seria que todas as máquinas de uma fábrica poderiam compartilhar Warehouse para assim evitar transferências de materiais dentro da fábrica.

Hoje é necessário alimentar os estoques das máquinas e em seguida transferir os produtos para Warehouse e não existe mecanismo para transferir semi-acabados de uma máquina para a próxima de uma linha de produção.

A linha de produção pode ter por exemplo, 10 máquinas tipo 1 no início, depois 5 máquinas tipo 2 em seguida, e mais 30 máquinas tipo 3 logo depois - o jogador deve planejar a demanda de acordo com a capacidade produtiva de cada máquina. Restrições de armazenamento não existem no jogo (ainda).
