# ADR-0010: Migrations com goose, em comando separado

- **Status:** Aceito
- **Data:** 2026-10-05

## Contexto

O schema do banco concentra boa parte das garantias do projeto: os índices únicos do ADR-0003, o `CHECK` de vagas do ADR-0004 e as restrições de webhooks e pagamentos do ADR-0005. Ele precisa evoluir de forma versionada, reproduzível e segura.

Requisitos:

- o schema fica em arquivos `.sql` que o `sqlc` consegue ler (ADR-0009);
- cada migration roda em uma transação, aproveitando o DDL transacional do PostgreSQL: uma migration que falha não deixa o schema pela metade;
- duas execuções simultâneas (dois deploys, duas instâncias) não podem aplicar a mesma migration;
- as migrations rodam a partir de um binário Go, sem instalar ferramentas no servidor.

## Decisão

- **Ferramenta:** `goose` (v3), usado como biblioteca, com os arquivos embutidos no binário via `embed`, e como CLI no desenvolvimento.
- **Arquivos:** em `migrations/`, um arquivo por migration com as seções `-- +goose Up` e `-- +goose Down`, numerados em sequência (`00001_create_events.sql`). A pasta é um pacote Go que exporta os arquivos com `//go:embed *.sql`.
- **Transação:** cada migration roda em transação (padrão do goose). Comandos que não podem rodar em transação, como `CREATE INDEX CONCURRENTLY`, ficam em um arquivo próprio marcado com `-- +goose NO TRANSACTION`.
- **Trava:** o goose usa um advisory lock do PostgreSQL para que só uma execução aplique migrations por vez.
- **Somente SQL:** migrations escritas em Go não são usadas. Correções de dados também são feitas em SQL, sem regra de negócio dentro de migrations.
- **Quando rodar:** um comando separado, `cmd/migrate`, executado no deploy antes de subir a API e o worker. A API não aplica migrations ao iniciar.
- **Down:** toda migration tem a seção `Down`, usada em desenvolvimento e verificada no CI. Em produção, correções são sempre feitas para a frente, com uma nova migration; o comando `down` do `cmd/migrate` exige uma opção explícita de ambiente de desenvolvimento.
- **Imutabilidade:** uma migration já commitada nunca é editada; qualquer mudança é uma nova migration.

## Alternativas consideradas

- **golang-migrate:** descartado. É o mais popular e usa SQL puro em arquivos `up`/`down` separados, mas não abre transação automaticamente: uma migration sem `BEGIN`/`COMMIT` que falha no meio deixa o banco marcado como "dirty", exigindo correção manual. Também traz muitas dependências por suportar dezenas de bancos.
- **tern:** descartado. Feito para PostgreSQL e pgx, com transação por padrão, mas tem comunidade menor, e seus templates permitem esconder SQL atrás de lógica.
- **Atlas (schema declarativo):** descartado. Calcula as mudanças a partir do estado desejado, o que esconde o SQL executado; é mais complexo e parte dos recursos é paga.
- **Aplicar migrations ao iniciar a API:** descartado. Uma migration com erro impediria a API de subir, e várias instâncias iniciando juntas disputariam a execução.
- **Não escrever migrations de Down:** descartado. Elas aceleram o desenvolvimento e o teste das próprias migrations, mesmo sem uso em produção.

## Consequências

- Uma migration que falha é desfeita por inteiro; o banco nunca fica em estado intermediário.
- O mesmo conjunto de arquivos alimenta o `sqlc` (que ignora as seções `Down`), os testes e a produção, então os testes rodam sobre o schema real.
- O deploy ganha um passo explícito: rodar `cmd/migrate` antes de atualizar a aplicação. Enquanto a versão antiga ainda estiver no ar, as migrations precisam ser compatíveis com ela (por exemplo, adicionar coluna antes de passar a usá-la, remover só depois).
- O CI aplica todas as migrations em um banco vazio e verifica o ciclo `up`, `down`, `up`.
- Os comentários `-- +goose` são uma pequena sintaxe específica da ferramenta.
