# ADR-0009: Acesso a dados com sqlc e pgx

- **Status:** Aceito
- **Data:** 2026-10-05

## Contexto

O ADR-0008 isolou o acesso a dados no adaptador `postgres` de cada módulo, atrás de interfaces de repositório. Falta escolher como esse adaptador fala com o PostgreSQL.

As regras dos ADRs 0003, 0004 e 0005 dependem de recursos específicos do PostgreSQL: índice único parcial, `UPDATE` condicional com contagem de linhas afetadas, `FOR UPDATE SKIP LOCKED` para os workers e o código de erro `23505` com o nome da restrição. A camada escolhida precisa expressar esses recursos com clareza e permitir passar a mesma transação entre módulos (ADR-0008, regra 4).

## Decisão

- **Driver:** `pgx/v5`, com `pgxpool` para o pool de conexões.
- **Consultas:** escritas em arquivos `.sql` dentro do adaptador `postgres` de cada módulo e convertidas em código Go tipado pelo `sqlc`. O `sqlc` lê o schema dos arquivos de migration e falha na geração se uma consulta citar coluna ou tipo inexistente.
- **Um pacote gerado por módulo**, configurado no `sqlc.yaml`. Os tipos gerados não saem do adaptador: o repositório os converte em entidades de domínio.
- **Transações:** o código gerado aceita a interface `DBTX`, satisfeita tanto pelo pool quanto por `pgx.Tx`; o repositório usa a `tx` recebida do caso de uso.
- **Consultas dinâmicas** (por exemplo, a busca do organizador com filtros opcionais) podem ser escritas com `pgx` diretamente, no mesmo adaptador.
- **Código gerado é versionado** no repositório, e o CI verifica que ele está atualizado (`sqlc diff`).

## Alternativas consideradas

- **pgx puro:** descartado. Sem geração de código, mas cada consulta exige leitura manual das colunas, e erros de SQL só aparecem em tempo de execução.
- **`database/sql` com pgx como driver:** descartado. É a biblioteca padrão, mas a independência de banco não traz ganho aqui, já que os ADRs usam recursos próprios do PostgreSQL, e perde recursos do pgx como `LISTEN/NOTIFY`.
- **ORM (GORM, ent):** descartado. Esconde o SQL do adaptador, mas o índice parcial, o `UPDATE` condicional e o `SKIP LOCKED` continuariam como SQL em strings ou em cláusulas especiais, ficando metade escondidos e metade expostos. Com o ADR-0008, o SQL já fica isolado das regras de negócio, que é o benefício que se buscaria com o ORM.

## Consequências

- Erros de nome de coluna e de tipo aparecem na geração do código, antes de rodar.
- O fluxo de desenvolvimento ganha um passo: alterar uma consulta exige rodar `sqlc generate`.
- Há código de conversão entre os tipos gerados e as entidades de domínio em cada repositório.
- A ferramenta de migrations (próximo ADR) precisa manter o schema em arquivos SQL que o `sqlc` consiga ler.
- O `sqlc` não impede que um módulo consulte a tabela de outro; essa regra continua verificada por revisão e testes de arquitetura.
