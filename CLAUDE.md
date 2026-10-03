# CLAUDE.md

Contexto do projeto para o Claude Code. Leia este arquivo inteiro antes de qualquer tarefa.

## O projeto

Plataforma de inscrições para eventos pequenos, começando por eventos esportivos (pedais, corridas, trilhas).

O projeto nasceu de um evento de ciclismo em que a plataforma usada falhou na gestão de participantes: pessoas que pagaram não apareciam na lista, havia participantes duplicados e a lista de largada não batia com as outras. Os pagamentos em si estavam corretos; o problema era a ligação entre pagamento e inscrição. Todo o design existe para tornar esses erros impossíveis.

Objetivo pessoal do autor: ganhar visão de arquitetura. Ao propor soluções, explique os trade-offs, não apenas o código.

### Escopo do MVP

- Organizador cria evento e categorias (nome, preço, vagas).
- Participante se inscreve informando os dados de todos os participantes no checkout e paga via Pix, por um único gateway.
- Participante recebe comprovante com o código do pedido e pode consultar o status.
- Organizador vê inscrições confirmadas, divergências de pagamento e gera a lista de largada.

Fora do MVP: cartão e boleto, lotes, cupons, transferência de inscrição, check-in com QR, tela de possíveis duplicados.

## Decisões de arquitetura

As decisões estão em `docs/adr/`. **Leia os ADRs antes de propor qualquer mudança estrutural.** Não contrarie um ADR aceito no código; se uma decisão precisar mudar, proponha um novo ADR usando `docs/adr/0000-template.md`.

Resumo:

- **ADR-0001:** monólito modular em Go, um único banco PostgreSQL.
- **ADR-0002:** pedido (transação, comprador, pagamento) é separado de inscrição (uma pessoa no evento).
- **ADR-0003:** unicidade garantida por restrições e índices do banco, não só pelo código.
- **ADR-0004:** vagas controladas por `UPDATE` atômico em `categoria`, sem Redis.
- **ADR-0005:** webhooks persistidos antes de processar, mais job de reconciliação com o gateway.

## Stack

- **Backend:** Go, PostgreSQL.
- **Ainda a decidir** (cada escolha deve virar um ADR antes de entrar no código): driver e camada de acesso a dados (por exemplo pgx, sqlc), roteador HTTP, ferramenta de migrations, gateway de pagamento, frontend.

Ao precisar de uma dessas escolhas, apresente 2 ou 3 opções com prós e contras e espere a decisão. Prefira a biblioteca padrão quando ela resolver bem o problema.

## Estrutura de código

Layout planejado (ajuste por ADR se mudar):

```
cmd/
  api/        # servidor HTTP
  worker/     # webhooks, reconciliação, expiração de pedidos, e-mails
internal/
  eventos/
  pedidos/
  inscricoes/
  pagamentos/
  notificacoes/
  platform/   # banco, config, logging, transações: infraestrutura compartilhada
migrations/
docs/adr/
```

Regras dos módulos:

- Cada módulo em `internal/<modulo>` expõe uma interface pública (tipos e funções exportados). Outros módulos usam apenas essa interface.
- Um módulo nunca lê ou escreve diretamente as tabelas de outro módulo.
- Dependências entre módulos não podem ser circulares. Se surgir a necessidade, sinalize: provavelmente a fronteira está errada.
- Operações que envolvem vários módulos na mesma transação (como confirmar pagamento) recebem a transação explicitamente, por exemplo um `pgx.Tx` ou uma interface equivalente vinda de `platform`.

## Regras de domínio que não podem ser quebradas

1. **Listas operacionais** (largada, kits, camisetas, premiação) são sempre consultas sobre inscrições com status `confirmada`. Nunca sobre pedidos, nunca de cópias ou caches.
2. **Estados.** Pedido: `aguardando_pagamento` → `pago` | `expirado`. Inscrição: `reservada` → `confirmada` | `cancelada`. Pagamento: `pendente` → `aprovado` | `recusado`. Toda transição grava uma linha em `historico` com autor e data.
3. **Nada é apagado.** Pedidos expirados e inscrições canceladas permanecem no banco.
4. **Nova tentativa de pagamento** cria um novo registro em `pagamento` ligado ao mesmo pedido, nunca um novo pedido.
5. **`confirmarPagamento` é idempotente** e é o único caminho para confirmar um pedido. Webhook e reconciliação chamam a mesma função. Ela roda em uma transação: se o pedido já está `pago`, não faz nada.
6. **Webhook:** validar assinatura, gravar em `webhook_recebido`, responder 200. O processamento acontece no worker, nunca no handler HTTP.
7. **Vagas:** reservadas com o `UPDATE ... WHERE vagas_ocupadas + n <= vagas` do ADR-0004, na mesma transação que cria pedido e inscrições.
8. **Número de peito:** atribuído uma única vez na confirmação, a partir do contador do evento; nunca reutilizado.
9. **Violação de restrição única** é resultado esperado: trate o erro do PostgreSQL (código `23505`) e devolva uma mensagem clara ao usuário.
10. **E-mails** são enfileirados na mesma transação (padrão outbox) e enviados pelo worker.

## Convenções

- Valores monetários sempre em centavos, tipo inteiro (`int64`). Nunca `float`.
- Datas e horas em `timestamptz`, armazenadas em UTC; conversão para o fuso do evento só na apresentação.
- Nomes de tabelas, colunas e status em português e `snake_case`, como nos ADRs.
- Erros com contexto (`fmt.Errorf("confirmar pagamento %d: %w", id, err)`); erros de domínio como valores comparáveis com `errors.Is`.
- `context.Context` como primeiro parâmetro de funções que fazem I/O.
- Código formatado com `gofmt` e sem avisos de `go vet`.

## Dados pessoais (LGPD)

CPF, data de nascimento e e-mail dos participantes são dados pessoais. Não registre esses dados em logs, não os inclua em mensagens de erro e não use dados reais em testes ou seeds. O acesso a eles é restrito ao organizador do evento.

## Testes

- Regras que dependem do banco (unicidade, vagas, concorrência, idempotência) são testadas contra um PostgreSQL real, não com mocks.
- Testes obrigatórios para:
  - o mesmo webhook processado duas vezes;
  - webhook e reconciliação confirmando o mesmo pedido ao mesmo tempo;
  - várias requisições disputando as últimas vagas de uma categoria;
  - duas inscrições simultâneas com o mesmo CPF no mesmo evento;
  - pagamento aprovado chegando para um pedido já expirado.
- Comandos: `go build ./...`, `go vet ./...`, `go test ./...`.

## Como trabalhar neste repositório

- Antes de uma tarefa grande, apresente um plano curto e espere confirmação.
- Mudanças pequenas e focadas; uma preocupação por commit.
- Mensagens de commit no formato `tipo: descrição` (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`).
- Se perceber que uma tarefa contradiz um ADR ou uma regra deste arquivo, pare e aponte o conflito em vez de contorná-lo.
