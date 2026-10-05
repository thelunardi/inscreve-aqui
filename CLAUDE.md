# CLAUDE.md

Contexto do projeto para o Claude Code. Leia este arquivo inteiro antes de qualquer tarefa.

## O projeto

Plataforma de inscrições para eventos pequenos, começando por eventos esportivos (pedais, corridas, trilhas).

O projeto nasceu de um evento de ciclismo em que a plataforma usada falhou na gestão de participantes: pessoas que pagaram não apareciam na lista, havia participantes duplicados e a lista de largada não batia com as outras. Os pagamentos em si estavam corretos; o problema era a ligação entre pagamento e inscrição. Todo o design existe para tornar esses erros impossíveis.

Objetivo pessoal do autor: ganhar visão de arquitetura e aplicar boas práticas de design (SOLID, padrões de projeto) de forma idiomática em Go. Ao propor soluções, explique os trade-offs, não apenas o código.

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
- **ADR-0006:** Go como linguagem do backend, preferindo a biblioteca padrão.
- **ADR-0007:** código e banco em inglês; documentação e mensagens ao usuário em português. O glossário do ADR traduz os nomes em português dos ADRs 0001–0005.
- **ADR-0008:** cada módulo em camadas (`domain`, `app`, `postgres`, `http`), com repositórios e transação explícita (Unit of Work).
- **ADR-0009:** acesso a dados com `pgx/v5` e `sqlc`, isolado no adaptador `postgres` de cada módulo.
- **ADR-0010:** migrations com `goose`, somente SQL, uma transação por migration, aplicadas por `cmd/migrate` antes do deploy.

## Stack

- **Backend:** Go, PostgreSQL.
- **Acesso a dados:** `pgx/v5` (`pgxpool`) e `sqlc`. Código gerado é versionado; rode `sqlc generate` ao alterar consultas.
- **Migrations:** `goose` (v3), arquivos `.sql` em `migrations/` embutidos no binário.
- **Ainda a decidir** (cada escolha deve virar um ADR antes de entrar no código): roteador HTTP, gateway de pagamento, frontend.

Ao precisar de uma dessas escolhas, apresente 2 ou 3 opções com prós e contras e espere a decisão. Prefira a biblioteca padrão quando ela resolver bem o problema.

## Estrutura de código

Layout planejado (ajuste por ADR se mudar):

```text
cmd/
  api/            # servidor HTTP (composition root)
  worker/         # webhooks, reconciliação, expiração de pedidos, e-mails
  migrate/        # aplica as migrations; roda no deploy, antes da API e do worker
internal/
  events/
  orders/
    orders.go     # API pública do módulo: fachada, tipos expostos, construtor
    internal/
      domain/     # entidades, transições de status, erros de domínio
      app/        # casos de uso e interfaces (portas) de que eles precisam
      postgres/   # repositórios, consultas .sql e código gerado pelo sqlc
      http/       # handlers e validação de entrada
  registrations/  # mesma estrutura interna em todos os módulos
  payments/
  notifications/
  platform/       # banco, Transactor, config, logging: infraestrutura compartilhada
migrations/       # 00001_create_events.sql...; pacote Go com //go:embed *.sql
docs/adr/
```

Regras dos módulos (ADR-0001 e ADR-0008):

- Outros módulos só importam o pacote raiz (`internal/<module>`). O miolo fica em `internal/<module>/internal/`, que o compilador protege.
- Um módulo nunca lê ou escreve diretamente as tabelas de outro módulo.
- Dependências entre módulos não podem ser circulares. Se surgir a necessidade, sinalize: provavelmente a fronteira está errada.
- Dependências apontam para dentro: `domain` não importa nada de infraestrutura; `app` importa `domain` e declara interfaces; `postgres` e `http` as implementam ou usam. Só `postgres` e `platform` importam `pgx`.
- SQL existe somente no pacote `postgres`. Os tipos gerados pelo sqlc não saem dele: o repositório os converte em entidades de domínio.
- Interfaces pequenas, declaradas por quem as usa (em `app`). Não crie interface para cada struct.
- Transações: `platform` oferece um `Transactor` (`WithinTx(ctx, func(ctx context.Context, tx db.Tx) error) error`). `db.Tx` é opaco e passado explicitamente aos repositórios; nunca carregue a transação no `context.Context`.
- Transições de status são métodos da entidade (`order.MarkPaid()`, `registration.Confirm()`) que devolvem erro de domínio quando a transição é inválida.
- Dependências montadas à mão em `cmd/api` e `cmd/worker`, sem framework de injeção.

## Regras de domínio que não podem ser quebradas

Nomes conforme o glossário do ADR-0007.

1. **Listas operacionais** (largada, kits, camisetas, premiação) são sempre consultas sobre `registrations` com status `confirmed`. Nunca sobre pedidos, nunca de cópias ou caches.
2. **Estados.** Pedido: `awaiting_payment` → `paid` | `expired`. Inscrição: `reserved` → `confirmed` | `canceled`. Pagamento: `pending` → `approved` | `rejected`. Toda transição grava uma linha em `status_history` com autor e data.
3. **Nada é apagado.** Pedidos expirados e inscrições canceladas permanecem no banco.
4. **Nova tentativa de pagamento** cria um novo registro em `payments` ligado ao mesmo pedido, nunca um novo pedido.
5. **`ConfirmPayment` é idempotente** e é o único caminho para confirmar um pedido. Webhook e reconciliação chamam a mesma função. Ela roda em uma transação: se o pedido já está `paid`, não faz nada.
6. **Webhook:** validar assinatura, gravar em `received_webhooks`, responder 200. O processamento acontece no worker, nunca no handler HTTP.
7. **Vagas:** reservadas com o `UPDATE categories SET slots_taken = slots_taken + n WHERE id = $1 AND slots_taken + n <= capacity` do ADR-0004, na mesma transação que cria pedido e inscrições.
8. **Número de peito (`bib_number`):** atribuído uma única vez na confirmação, a partir do contador do evento; nunca reutilizado.
9. **Violação de restrição única** é resultado esperado: o repositório traduz o erro do PostgreSQL (`23505`), pelo nome da restrição, em um erro de domínio (ex.: `ErrCPFAlreadyRegistered`), e a camada `http` mostra uma mensagem clara em português.
10. **E-mails** são enfileirados na mesma transação em `outbox_messages` e enviados pelo worker.

## Convenções

- Migrations (ADR-0010): um arquivo por mudança com seções `-- +goose Up` e `-- +goose Down`, somente SQL, numeração sequencial. Nunca edite uma migration já commitada; crie uma nova. Em produção, corrija para a frente. Mantenha cada migration compatível com a versão da aplicação que ainda está no ar.
- Valores monetários sempre em centavos, tipo inteiro (`int64`). Nunca `float`.
- Datas e horas em `timestamptz`, armazenadas em UTC; conversão para o fuso do evento só na apresentação.
- Código e banco em inglês (ADR-0007): pacotes, arquivos, tipos, funções, tabelas (no plural), colunas e status. Banco em `snake_case`. `cpf` e `pix` ficam como estão.
- Documentação, ADRs, mensagens ao usuário e e-mails em português.
- Erros com contexto (`fmt.Errorf("confirm payment %d: %w", id, err)`); erros de domínio como valores comparáveis com `errors.Is`.
- `context.Context` como primeiro parâmetro de funções que fazem I/O.
- Código formatado com `gofmt` e sem avisos de `go vet`.

## Dados pessoais (LGPD)

CPF, data de nascimento e e-mail dos participantes são dados pessoais. Não registre esses dados em logs, não os inclua em mensagens de erro e não use dados reais em testes ou seeds. O acesso a eles é restrito ao organizador do evento.

## Testes

- Regras de domínio (transições de status, cálculos) têm testes de unidade em `domain`, sem banco.
- Regras que dependem do banco (unicidade, vagas, concorrência, idempotência) são testadas contra um PostgreSQL real, não com mocks.
- Um teste de arquitetura verifica que `domain` e `app` não importam `pgx` nem pacotes de outras camadas externas.
- Testes obrigatórios para:
  - o mesmo webhook processado duas vezes;
  - webhook e reconciliação confirmando o mesmo pedido ao mesmo tempo;
  - várias requisições disputando as últimas vagas de uma categoria;
  - duas inscrições simultâneas com o mesmo CPF no mesmo evento;
  - pagamento aprovado chegando para um pedido já expirado.
- Comandos: `go build ./...`, `go vet ./...`, `go test ./...`.

## Como trabalhar neste repositório

- Responda ao autor em português.
- Antes de uma tarefa grande, apresente um plano curto e espere confirmação.
- Mudanças pequenas e focadas; uma preocupação por commit.
- Mensagens de commit no formato `tipo: descrição` (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`).
- Se perceber que uma tarefa contradiz um ADR ou uma regra deste arquivo, pare e aponte o conflito em vez de contorná-lo.
