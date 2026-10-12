# ADR-0011: Cada tabela pertence a um módulo; histórico de status por módulo

- **Status:** Aceito
- **Data:** 2026-10-12

## Contexto

O ADR-0003 determina que toda mudança de status grave uma linha em uma única tabela `historico` (no ADR-0007, `status_history`). Ao mesmo tempo, os ADRs 0001 e 0008 dizem que um módulo nunca lê ou escreve as tabelas de outro. Três módulos mudam status (`orders`, `registrations`, `payments`), então uma tabela única de histórico seria escrita por todos eles, sem um dono.

Também é preciso decidir se as tabelas de módulos diferentes podem se referenciar por chaves estrangeiras, já que o ADR-0003 quer a integridade garantida pelo banco e o ADR-0001 quer fronteiras claras entre módulos.

## Decisão

**Cada tabela tem exatamente um módulo dono**, o único que a lê e escreve pelo código:

| Módulo | Tabelas |
| --- | --- |
| `events` | `events`, `categories` |
| `orders` | `orders`, `order_status_history` |
| `registrations` | `registrations`, `registration_status_history` |
| `payments` | `payments`, `payment_status_history`, `received_webhooks` |
| `notifications` | `outbox_messages` |

**O histórico de status é uma tabela por módulo**, com chave estrangeira para a entidade de origem, os status anterior e novo, o autor e a data. Isso substitui a tabela única `historico` do ADR-0003 (e `status_history` no glossário do ADR-0007); a regra de registrar toda transição continua igual.

**Chaves estrangeiras entre módulos são permitidas** (por exemplo `registrations.order_id` → `orders.id`). A regra dos módulos trata do acesso pelo código; a chave estrangeira é uma garantia do banco, que nenhum código precisa ler. Quando um módulo precisa de dados ou de uma alteração em tabela de outro (reservar vagas em `categories`, enfileirar um e-mail), ele chama a API pública do módulo dono, passando a mesma transação.

## Alternativas consideradas

- **Uma única `status_history` de responsabilidade do `platform`:** descartado. Segue o ADR-0003 ao pé da letra, mas a tabela fica genérica (`entity_type`, `entity_id`) e sem chave estrangeira, então o banco não garante que a entidade citada existe.
- **Sem chaves estrangeiras entre módulos:** descartado. Facilitaria extrair um módulo como serviço, mas abriria mão de garantias do banco, como impedir inscrição para um pedido inexistente, sem motivo concreto no MVP.

## Consequências

- Cada histórico tem integridade referencial e evolui junto com a entidade do seu módulo.
- Ver tudo o que aconteceu com um pedido (pedido, inscrições e pagamentos) exige juntar três tabelas de histórico; essa consulta é de leitura e pode ser montada combinando as APIs dos três módulos.
- Extrair um módulo para um serviço separado no futuro exigirá remover as chaves estrangeiras que cruzam a fronteira; isso será registrado no ADR da extração.
- Toda tabela nova precisa ter o seu dono definido nesta tabela ou em um ADR posterior.
