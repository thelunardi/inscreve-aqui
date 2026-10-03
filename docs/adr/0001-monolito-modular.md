# ADR-0001: Monólito modular em vez de microserviços

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

O MVP atende eventos pequenos, com centenas de inscrições, e será desenvolvido por uma pessoa. Os problemas que o sistema precisa resolver são de consistência de dados entre pedido, pagamento e inscrição. Distribuir esses dados entre serviços tornaria a consistência mais difícil, não mais fácil.

## Decisão

O backend será uma única aplicação com um único banco PostgreSQL, dividida em módulos com fronteiras explícitas: `eventos`, `pedidos`, `inscricoes`, `pagamentos` e `notificacoes`. Um módulo só acessa outro pela sua interface pública, nunca pelas tabelas diretamente. Tarefas assíncronas (e-mails, processamento de webhooks, reconciliação) rodam como workers do mesmo processo ou de um segundo processo com o mesmo código.

## Alternativas consideradas

- **Microserviços desde o início:** descartado. Exigiria transações distribuídas ou sagas para confirmar pagamento e inscrição juntos, além de deploy e observabilidade de vários serviços, sem ganho real nesse volume.
- **Monólito sem módulos definidos:** descartado. Funciona no começo, mas impede extrair um serviço no futuro e mistura regras de negócio.

## Consequências

- Confirmar pagamento, inscrições e número de peito cabe em uma única transação do banco.
- Um único deploy e um único banco para operar.
- As fronteiras dos módulos precisam de disciplina; vale verificar dependências proibidas com testes de arquitetura.
- Candidatos naturais a virar serviço no futuro: `pagamentos` e um check-in offline. A extração deve ter um motivo registrado em novo ADR.
