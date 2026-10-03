# Inscrições

Plataforma de inscrições para eventos pequenos, começando por eventos esportivos como pedais, corridas e trilhas.

## Por que este projeto existe

Em um evento de ciclismo organizado com uma plataforma de terceiros, a gestão dos participantes falhou:

- pessoas que pagaram não apareciam na lista de participantes;
- havia participantes duplicados;
- a lista de largada não batia com as outras listas do evento.

Os pagamentos em si estavam corretos. O problema estava na ligação entre pagamento e inscrição e na falta de uma fonte única da verdade para as listas. Este projeto é construído para que esses problemas sejam impossíveis por design.

## Decisões de arquitetura

As decisões estruturais estão registradas como ADRs (Architecture Decision Records) em [`docs/adr`](docs/adr/README.md).

## Status

Em planejamento. MVP: organizador cria o evento e as categorias, participante se inscreve e paga via Pix, recebe comprovante com código, e o organizador gera a lista de largada a partir das inscrições confirmadas.
