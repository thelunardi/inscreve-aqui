# ADR-0002: Separar pedido de inscrição

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

No evento de ciclismo que motivou o projeto, participantes que pagaram não apareciam na lista e havia participantes duplicados. Uma causa provável é tratar comprador e participante como a mesma coisa: quem paga por três pessoas aparece uma vez, e os outros dois somem ou aparecem de novo em outra compra.

## Decisão

O modelo terá duas entidades distintas:

- **Pedido:** a transação. Guarda o comprador, o valor, o prazo de expiração e o status do pagamento (`aguardando_pagamento`, `pago`, `expirado`). Um pedido pode ter vários pagamentos, por exemplo um Pix expirado e um novo.
- **Inscrição:** uma pessoa no evento. Guarda nome, CPF, data de nascimento, categoria, camiseta e número de peito, com status próprio (`reservada`, `confirmada`, `cancelada`).

Um pedido gera uma ou mais inscrições. Todas as listas operacionais (largada, kits, camisetas, premiação) são consultas sobre inscrições confirmadas, nunca sobre pedidos. No MVP, os dados de todos os participantes são obrigatórios no checkout.

## Alternativas consideradas

- **Uma entidade única "inscrição com pagamento":** descartada. Não representa compras para terceiros e faz cada nova tentativa de pagamento virar uma nova inscrição.
- **Coletar os dados dos participantes depois da compra:** adiado. Reduz o atrito no checkout, mas cria o estado "pago sem dados", que é justamente o tipo de inscrição que some das listas.

## Consequências

- Quando o pedido vira `pago`, todas as suas inscrições viram `confirmada` na mesma transação.
- A busca do organizador procura nos dois lados: o nome de alguém encontra a inscrição dela e os pedidos que ela pagou para outras pessoas.
- Nada é apagado: pedidos expirados e inscrições canceladas permanecem no banco para auditoria.
- Transferência de inscrição para outra pessoa passa a ser uma operação simples sobre a inscrição, sem mexer no pedido.
