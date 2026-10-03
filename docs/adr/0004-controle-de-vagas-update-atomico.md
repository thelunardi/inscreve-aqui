# ADR-0004: Controle de vagas por UPDATE atômico no PostgreSQL

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

Cada categoria tem um limite de vagas. Quando várias pessoas tentam se inscrever nas últimas vagas ao mesmo tempo, o sistema não pode aceitar mais inscrições do que existem. A vaga precisa ficar reservada enquanto a pessoa paga e voltar ao estoque se o pagamento não acontecer.

## Decisão

A tabela `categoria` guarda `vagas` e `vagas_ocupadas` (reservadas mais confirmadas). A reserva é um único comando atômico, na mesma transação que cria o pedido e as inscrições:

```sql
UPDATE categoria
   SET vagas_ocupadas = vagas_ocupadas + $2
 WHERE id = $1 AND vagas_ocupadas + $2 <= vagas;
```

Zero linhas afetadas significa vagas esgotadas, e a transação é desfeita. Uma restrição `CHECK (vagas_ocupadas <= vagas)` protege contra qualquer outro caminho. Um job periódico expira pedidos vencidos: marca o pedido como `expirado`, as inscrições como `cancelada` e devolve as vagas, tudo em uma transação.

## Alternativas consideradas

- **`SELECT ... FOR UPDATE` seguido de verificação e update:** descartado. Correto, mas mais código e mais tempo segurando o lock.
- **Contador no Redis com TTL para a reserva:** descartado para o MVP. Escala melhor sob picos grandes, mas cria uma segunda fonte da verdade que precisa ser mantida em sincronia com o banco.
- **Contar inscrições ativas a cada tentativa (`COUNT(*)`):** descartado. Sofre da mesma condição de corrida da validação no código.

## Consequências

- Overbooking fica impossível sem nenhuma infraestrutura além do PostgreSQL.
- A linha de cada categoria vira um ponto de contenção. Para eventos pequenos isso é irrelevante; se um teste de carga mostrar gargalo na abertura de vendas, a troca por Redis será um novo ADR com os números do teste.
- O prazo de reserva (sugestão: 30 minutos para Pix) precisa ser maior que o prazo de expiração da cobrança no gateway, para que uma vaga nunca seja liberada enquanto o Pix ainda pode ser pago.
