# ADR-0006: Go como linguagem do backend

- **Status:** Aceito
- **Data:** 2026-10-03

## Contexto

O ADR-0001 definiu um monólito modular com um único banco relacional, mas não escolheu a linguagem. O backend precisa de:

- transações explícitas atravessando módulos (confirmar pagamento, reservar vagas), com controle claro de quando começam e terminam;
- workers de longa duração no mesmo código (webhooks, reconciliação, expiração de pedidos, e-mails), rodando com concorrência;
- fronteiras de módulo que possam ser verificadas, não apenas combinadas;
- deploy simples, operado por uma pessoa.

O projeto também é um exercício de arquitetura para o autor: a linguagem deve deixar as decisões visíveis no código, em vez de escondê-las atrás de um framework.

## Decisão

O backend será escrito em Go, usando a biblioteca padrão sempre que ela resolver bem o problema. Bibliotecas externas entram por ADR próprio (acesso a dados, migrations, roteador HTTP).

- Cada módulo é um pacote em `internal/<modulo>`; o compilador impede importações de fora do módulo raiz, e importações circulares entre pacotes não compilam.
- API e worker são dois binários (`cmd/api`, `cmd/worker`) gerados do mesmo código.
- Transações são passadas explicitamente como parâmetro, sem gerenciamento implícito por anotação ou contexto global.

## Alternativas consideradas

- **TypeScript (Node.js):** descartado. Ecossistema grande e mesmo idioma no frontend, mas a concorrência dos workers depende do event loop, e os ORMs mais comuns tendem a esconder transações e SQL, que são o centro deste projeto.
- **Java ou Kotlin (Spring):** descartado. Maduro e com transações bem resolvidas, mas o gerenciamento por anotação (`@Transactional`) esconde justamente as fronteiras que o projeto quer tornar explícitas, e o framework pesa para uma pessoa só.
- **Python (Django ou FastAPI):** descartado. Produtivo, mas a tipagem é opcional e a concorrência dos workers exige peças extras (Celery, asyncio).

## Consequências

- Um binário estático por processo, sem runtime para instalar; deploy e imagem de contêiner pequenos.
- Goroutines atendem os workers e os testes de concorrência exigidos (vagas, CPF duplicado, webhook e reconciliação simultâneos) sem infraestrutura extra.
- Go não tem ORM dominante nem framework padrão; cada peça (acesso a dados, migrations, HTTP) precisa ser escolhida, o que gera os próximos ADRs.
- A regra "um módulo não acessa as tabelas de outro" continua dependendo de disciplina; o compilador protege importações circulares, mas não o SQL. Testes de arquitetura continuam recomendados (ADR-0001).
- O frontend fica em aberto e pode usar outra linguagem.
