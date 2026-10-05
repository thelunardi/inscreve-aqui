# ADR-0007: Nomes em inglês no código e no banco de dados

- **Status:** Aceito
- **Data:** 2026-10-05

## Contexto

Os ADRs 0002 a 0005 e o CLAUDE.md usaram nomes em português para tabelas, colunas, status e funções (`pedido`, `inscricao`, `aguardando_pagamento`, `confirmarPagamento`). Nenhum código foi escrito ainda.

Go, suas bibliotecas e a maior parte da documentação técnica estão em inglês. Misturar idiomas no código produz nomes como `GetPedidoByID` ou `pedido.IsPago()`, e o `sqlc` gera nomes de structs a partir das tabelas, então um banco em português levaria o português para dentro do código gerado.

## Decisão

- **Em inglês:** pacotes, arquivos, tipos, funções, variáveis, tabelas, colunas, índices, restrições e valores de status.
- **Em português:** documentação (README, ADRs, CLAUDE.md), mensagens exibidas ao usuário e textos de e-mail.
- **Termos brasileiros sem tradução fiel** mantêm o nome original: `cpf`, `pix`.
- **Tabelas no plural** (`orders`, `registrations`). Além de ser a convenção comum, evita `order`, que é palavra reservada do SQL. O `sqlc` gera as structs no singular (`Order`, `Registration`).

Os ADRs aceitos não são reescritos; seus nomes em português são lidos pelo glossário abaixo.

### Glossário

| Conceito (ADRs 0001–0005) | Código e banco |
| --- | --- |
| módulos `eventos`, `pedidos`, `inscricoes`, `pagamentos`, `notificacoes` | `events`, `orders`, `registrations`, `payments`, `notifications` |
| evento / categoria | `events` / `categories` |
| vagas / vagas ocupadas | `capacity` / `slots_taken` |
| pedido / comprador | `orders` / `buyer` |
| chave de idempotência | `idempotency_key` |
| inscrição | `registrations` |
| data de nascimento / camiseta | `birth_date` / `shirt_size` |
| número de peito | `bib_number` |
| pagamento / id externo | `payments` / `external_id` |
| webhook recebido / id do evento externo | `received_webhooks` / `external_event_id` |
| histórico | `status_history` |
| outbox de e-mails | `outbox_messages` |
| lista de largada | start list |
| divergências | discrepancies |
| `confirmarPagamento` | `ConfirmPayment` |

| Status (ADRs 0001–0005) | Código e banco |
| --- | --- |
| pedido: `aguardando_pagamento`, `pago`, `expirado` | `awaiting_payment`, `paid`, `expired` |
| inscrição: `reservada`, `confirmada`, `cancelada` | `reserved`, `confirmed`, `canceled` |
| pagamento: `pendente`, `aprovado`, `recusado` | `pending`, `approved`, `rejected` |
| webhook: `recebido`, `erro` | `received`, `processed`, `failed` |

Exemplo, o índice do ADR-0003 fica:

```sql
CREATE UNIQUE INDEX uq_registrations_event_cpf
  ON registrations (event_id, cpf)
  WHERE status <> 'canceled';
```

## Alternativas consideradas

- **Tudo em português:** descartado. Mistura idiomas com a biblioteca padrão e com o código gerado, e afasta o projeto das convenções da comunidade Go.
- **Código em inglês e banco em português:** descartado. Exige configurar no `sqlc` um nome em inglês para cada tabela e coluna, e as consultas SQL ficariam em um idioma enquanto o código ao redor fica em outro.

## Consequências

- Código, banco e código gerado usam o mesmo vocabulário, sem tradução entre camadas.
- O glossário passa a ser a referência de nomes; um conceito novo entra nele antes de entrar no código.
- Quem lê os ADRs 0001–0005 precisa do glossário para achar os nomes no código.
- Mensagens ao usuário continuam em português, então os erros de domínio (em inglês) são traduzidos na camada HTTP.
