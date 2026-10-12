# Inscrições

[![CI](https://github.com/thelunardi/inscreve-aqui/actions/workflows/ci.yml/badge.svg)](https://github.com/thelunardi/inscreve-aqui/actions/workflows/ci.yml)

Plataforma de inscrições para eventos pequenos, começando por eventos esportivos como pedais, corridas e trilhas.

## Por que este projeto existe

Em um evento de ciclismo organizado com uma plataforma de terceiros, a gestão dos participantes falhou:

- pessoas que pagaram não apareciam na lista de participantes;
- havia participantes duplicados;
- a lista de largada não batia com as outras listas do evento.

Os pagamentos em si estavam corretos. O problema estava na ligação entre pagamento e inscrição e na falta de uma fonte única da verdade para as listas. Este projeto é construído para que esses problemas sejam impossíveis por design.

## Decisões de arquitetura

As decisões estruturais estão registradas como ADRs (Architecture Decision Records) em [`docs/adr`](docs/adr/README.md).

## Como rodar localmente

Pré-requisitos: Go 1.25 ou mais novo, Docker com Compose e `make`.

```sh
git clone git@github.com:thelunardi/inscreve-aqui.git
cd inscreve-aqui
cp .env.example .env   # variáveis de ambiente locais (não versionado)
make db-up             # sobe o PostgreSQL 18 em localhost:5432
make migrate-up        # aplica as migrations
make check             # formatação, go.mod, build, vet e testes
```

Os testes rodam contra o PostgreSQL local: cada teste cria um banco temporário e o apaga ao terminar, então não interferem nos dados do banco de desenvolvimento.

Outros comandos úteis:

| Comando | O que faz |
| --- | --- |
| `make help` | Lista todos os comandos |
| `make migrate-status` | Mostra as migrations aplicadas e pendentes |
| `make migration name=create_x` | Cria uma migration nova, já numerada |
| `make migrate-down` | Desfaz a última migration (só com `APP_ENV=development`) |
| `make test-race` | Roda os testes com o detector de condições de corrida, como no CI |
| `make db-reset` | Apaga os dados do PostgreSQL local e sobe de novo |
| `make db-down` | Para o PostgreSQL local, mantendo os dados |

## Status

Em desenvolvimento. Pronto: schema do banco com as garantias dos ADRs, migrations e CI. MVP: organizador cria o evento e as categorias, participante se inscreve e paga via Pix, recebe comprovante com código, e o organizador gera a lista de largada a partir das inscrições confirmadas.
