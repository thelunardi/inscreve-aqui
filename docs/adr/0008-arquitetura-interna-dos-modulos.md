# ADR-0008: Arquitetura interna dos módulos em camadas, com repositórios

- **Status:** Aceito
- **Data:** 2026-10-05

## Contexto

O ADR-0001 dividiu o monólito em módulos (com os nomes do ADR-0007: `events`, `orders`, `registrations`, `payments`, `notifications`), mas não definiu como cada módulo se organiza por dentro. Sem essa definição, regras de negócio, SQL e HTTP tendem a se misturar no mesmo arquivo, o que dificulta testar as regras, trocar detalhes de infraestrutura e revisar mudanças.

O projeto quer aplicar boas práticas de design (SOLID, padrões de projeto) de forma idiomática em Go: sem classes nem herança, com interfaces pequenas e composição.

## Decisão

Cada módulo segue uma arquitetura em camadas no estilo *ports and adapters* (hexagonal):

```text
internal/orders/
  orders.go           # API pública do módulo: fachada, tipos expostos e construtor
  internal/
    domain/           # entidades, transições de status, erros de domínio
    app/              # casos de uso e as interfaces (portas) de que eles precisam
    postgres/         # adaptador de banco: repositórios, SQL e código gerado
    http/             # adaptador HTTP: handlers e validação de entrada
```

Regras:

1. **Dependências apontam para dentro.** `domain` não importa nenhuma outra camada nem pacote de infraestrutura. `app` importa `domain` e define interfaces. `postgres` e `http` implementam ou usam essas interfaces. Nenhuma camada importa `pgx`, exceto `postgres` e `platform`.
2. **Fronteira do módulo garantida pelo compilador.** Tudo fica sob `internal/<module>/internal/`, que o Go impede de ser importado por outros módulos. Outro módulo só enxerga o pacote raiz (`orders.go`), que é a interface pública exigida pelo ADR-0001.
3. **Repository.** Casos de uso acessam dados por interfaces declaradas em `app`, pequenas e definidas por quem as usa (por exemplo `OrderRepository` com apenas os métodos necessários). O SQL existe somente no pacote `postgres`.
4. **Unit of Work para transações.** `platform` oferece um `Transactor` com `WithinTx(ctx, func(ctx context.Context, tx db.Tx) error) error`. `db.Tx` é um tipo opaco: os casos de uso o recebem e repassam aos repositórios, mas só os adaptadores `postgres` sabem extrair o `pgx.Tx`. Operações entre módulos, como `ConfirmPayment`, recebem a mesma `tx` explicitamente, como já exige o CLAUDE.md.
5. **Transições de status no domínio.** Cada entidade expõe métodos de transição (`order.MarkPaid()`, `registration.Confirm()`) que validam o estado atual e devolvem um erro de domínio quando a transição é inválida. O equivalente ao padrão State fica nesses métodos e em uma tabela de transições permitidas, sem um tipo por estado.
6. **Erros de banco traduzidos no adaptador.** O repositório converte a violação de restrição única (`23505`), pelo nome da restrição, em um erro de domínio (`registrations.ErrCPFAlreadyRegistered`). Nenhuma camada acima conhece códigos do PostgreSQL. A camada `http` traduz o erro de domínio para a mensagem em português exibida ao usuário.
7. **Composição manual.** As dependências são montadas em `cmd/api` e `cmd/worker` (composition root), sem framework de injeção de dependência.

## Alternativas consideradas

- **Uma camada por módulo (handler acessando o banco direto):** descartado. Menos arquivos, mas as regras de negócio ficam presas ao HTTP e ao SQL e só podem ser testadas de ponta a ponta.
- **Clean Architecture completa, com DTOs e mapeadores em cada fronteira:** descartado. Para o tamanho do projeto, a quantidade de conversões entre estruturas quase idênticas custa mais do que protege.
- **Transação carregada no `context.Context`:** descartado. É comum em Go, mas esconde quais operações participam da transação; o projeto prefere a `tx` como parâmetro visível.
- **Framework de injeção de dependência (wire, fx):** descartado. A montagem manual cabe em poucas dezenas de linhas e não esconde nada.
- **Um tipo por estado (padrão State clássico):** descartado. Em Go, métodos de transição sobre a própria entidade expressam o mesmo com menos código.

## Consequências

- As regras de domínio (transições de status, cálculo de valores) são testadas sem banco, com testes de unidade rápidos.
- As regras que dependem do banco (unicidade, vagas, concorrência, idempotência) continuam testadas contra um PostgreSQL real, agora nos testes do adaptador `postgres`.
- Trocar a camada de acesso a dados afeta apenas os pacotes `postgres`.
- Há mais pacotes e alguma conversão entre os tipos gerados pelo banco e as entidades de domínio.
- O pacote raiz de cada módulo precisa expor só o necessário; crescer essa fachada sem critério recria o acoplamento que a regra 2 evita.
- Um teste de arquitetura deve verificar as regras 1 e 2 (por exemplo, que `domain` e `app` não importam `pgx`), já que o compilador não verifica todas elas.
