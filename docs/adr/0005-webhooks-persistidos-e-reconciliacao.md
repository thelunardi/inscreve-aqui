# ADR-0005: Webhooks persistidos e job de reconciliação de pagamentos

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

O gateway avisa que um Pix foi pago por webhook. Esse aviso pode chegar duplicado, fora de ordem, atrasado ou nunca chegar; o servidor também pode estar fora do ar ou falhar ao processá-lo. Se o sistema depender só do webhook, um pagamento aprovado pode deixar a inscrição pendente para sempre, que é o sintoma "paguei e não estou na lista".

## Decisão

A confirmação de pagamento tem dois caminhos independentes que chamam a mesma função idempotente, `confirmarPagamento(pagamento)`:

1. **Webhook persistido.** O endpoint valida a assinatura do gateway, grava o payload em `webhook_recebido` com status `recebido` e responde 200 imediatamente. Um worker processa os registros pendentes, com novas tentativas e backoff; após várias falhas o registro fica com status `erro` e aparece para revisão.
2. **Reconciliação periódica.** A cada poucos minutos, um job consulta no gateway o status dos pagamentos de pedidos em `aguardando_pagamento` e chama a mesma função para os aprovados. Uma vez por dia, compara todos os pagamentos do evento no gateway com os pedidos pagos no banco.

`confirmarPagamento` roda em uma transação: se o pedido já está `pago`, não faz nada; caso contrário marca o pagamento como `aprovado`, o pedido como `pago`, as inscrições como `confirmada`, atribui os números de peito e grava o histórico. O e-mail de confirmação é enfileirado na mesma transação (padrão outbox) e enviado depois por um worker.

Se um pagamento aprovado chegar para um pedido já `expirado`, o sistema tenta reativá-lo se ainda houver vaga; se não houver, registra o caso para estorno e avisa o organizador.

## Alternativas consideradas

- **Processar o webhook de forma síncrona e confiar só nele:** descartado. É exatamente o modelo que perde confirmações.
- **Apenas polling no gateway, sem webhook:** descartado. Funciona, mas confirma com atraso e consome mais chamadas à API do gateway.
- **Fila externa (RabbitMQ, SQS) para os webhooks:** adiado. A tabela no PostgreSQL dá a mesma garantia de durabilidade no volume do MVP.

## Consequências

- Nenhum pagamento aprovado fica sem inscrição confirmada por mais que o intervalo da reconciliação.
- O organizador ganha uma tela de divergências: pagamentos sem inscrição confirmada, webhooks com erro, pagamentos em pedidos expirados.
- O participante recebe o código do pedido no comprovante e pode consultar o status sozinho.
- Testar a idempotência vira obrigatório: o mesmo webhook processado duas vezes, e webhook e reconciliação confirmando o mesmo pedido ao mesmo tempo.
- O sistema depende de o gateway oferecer consulta de pagamentos por API, o que deve ser critério na escolha do gateway.
