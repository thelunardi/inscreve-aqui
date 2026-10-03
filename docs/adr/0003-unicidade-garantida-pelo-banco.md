# ADR-0003: Unicidade e integridade garantidas pelo banco

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

Verificar duplicidade só no código ("existe inscrição com este CPF?" seguido de um insert) falha quando duas requisições chegam ao mesmo tempo, por exemplo por duplo clique ou recarregamento da página. As duas leem "não existe" e as duas inserem.

## Decisão

As regras de unicidade ficam no PostgreSQL, como restrições e índices:

- **Um CPF ativo por evento:** índice único parcial em `inscricao (evento_id, cpf) WHERE status <> 'cancelada'`. Inscrições canceladas não impedem uma nova.
- **Número de peito único por evento:** índice único em `inscricao (evento_id, numero_peito)`. O número sai de um contador do evento, incrementado na transação de confirmação, e nunca é reutilizado.
- **Envio do formulário idempotente:** o frontend gera uma `chave_idempotencia` por tentativa de checkout, com restrição única em `pedido`. Um reenvio devolve o pedido já criado em vez de criar outro.
- **Pagamento e webhook únicos:** restrições em `pagamento (gateway, id_externo)` e `webhook_recebido (gateway, id_evento_externo)`.
- **Histórico:** toda mudança de status grava uma linha na tabela `historico`, com autor e data.

O código trata a violação de restrição como um resultado esperado e devolve uma mensagem clara ("este CPF já está inscrito neste evento").

```sql
CREATE UNIQUE INDEX uq_inscricao_cpf_evento
  ON inscricao (evento_id, cpf)
  WHERE status <> 'cancelada';

CREATE UNIQUE INDEX uq_numero_peito
  ON inscricao (evento_id, numero_peito)
  WHERE numero_peito IS NOT NULL;
```

## Alternativas consideradas

- **Validação apenas na aplicação:** descartada pela condição de corrida descrita acima.
- **Lock na aplicação ou no Redis antes de inserir:** descartado. Funciona, mas é mais código para manter e falha se alguém inserir dados por outro caminho.
- **Unicidade por (evento, categoria, CPF):** adiada. Só é necessária se uma pessoa puder participar de duas provas no mesmo evento; pode virar um novo ADR.

## Consequências

- Duplicados por concorrência ficam impossíveis, não apenas improváveis.
- O CPF passa a ser obrigatório e validado no formulário. Por ser dado pessoal, entra nas regras da LGPD: acesso restrito ao organizador do evento e política de retenção após o evento.
- Duplicados que escapam (CPF digitado errado) continuam possíveis; uma tela de "possíveis duplicados" fica para depois do MVP.
