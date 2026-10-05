# Architecture Decision Records

Um ADR registra uma decisão importante: o contexto, a escolha feita, as alternativas descartadas e as consequências. ADRs não são editados depois de aceitos; se a decisão mudar, escreve-se um novo ADR que substitui o anterior, e o antigo recebe o status "Substituído por ADR-XXXX".

Para criar um novo ADR, copie [`0000-template.md`](0000-template.md) com o próximo número e abra um pull request.

| ADR | Decisão | Status |
| --- | --- | --- |
| [0001](0001-monolito-modular.md) | Monólito modular em vez de microserviços | Aceito |
| [0002](0002-separar-pedido-de-inscricao.md) | Separar pedido de inscrição | Aceito |
| [0003](0003-unicidade-garantida-pelo-banco.md) | Unicidade e integridade garantidas pelo banco | Aceito |
| [0004](0004-controle-de-vagas-update-atomico.md) | Controle de vagas por UPDATE atômico no PostgreSQL | Aceito |
| [0005](0005-webhooks-persistidos-e-reconciliacao.md) | Webhooks persistidos e job de reconciliação de pagamentos | Aceito |
| [0006](0006-linguagem-go.md) | Go como linguagem do backend | Aceito |
| [0007](0007-nomes-em-ingles.md) | Nomes em inglês no código e no banco de dados | Aceito |
