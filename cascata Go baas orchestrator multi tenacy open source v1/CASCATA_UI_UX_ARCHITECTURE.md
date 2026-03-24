# Arquitetura de UI/UX e Mapa de Experiência do Cascata Studio v1.0

Após uma análise profunda do MVP (v0) e de toda a fundação de código estabelecida no **Cascata Studio**, este documento redefine a arquitetura de frontend. O objetivo não é criar um painel administrativo comum, mas uma **Power Tool** de nível corporativo, focada em performance, redução de carga cognitiva e fluxos de trabalho do tipo "IDE in Browser" (Ambiente de Desenvolvimento Integrado).

---

## 1. Filosofia "Pré-Cognitiva" e Arquitetura Base

A UI do Cascata não deve interromper o estado de fluxo (*flow*) do arquiteto de software. 

### 1.1 Sistema de Ambientes (Draft vs Live)
A pedra angular da experiência do Cascata é o isolamento seguro de infraestrutura:
- **Environment Switcher:** Um componente global na Sidebar Inferior controlando o estado de `live` (produção) e `draft` (teste/desenvolvimento).
- **Draft Creation & Rebase Strategy:** Permite criar clones estáticos (Data Clone) em porcentagens (0% a 100%). Se um *draft* conflitar com o *live*, um Modal de Rebase oferece "Resume Work" ou "Rebase from Live" (Overwrite).
- **Integração de Rede (API):** Um Global Fetch Interceptor injeta passivamente o header `x-cascata-env`, garantindo que toda a interface transacione com o banco de dados correto sem recarregar a tela ou poluir os componentes com verificações condicionais de ambiente.

### 1.2 Navegação Híbrida (Sidebar + Quick Peek)
Navegar quebra o contexto. Para mitigar isso:
- **Menu Dinâmico (Drag & Drop):** Sidebar com capacidade de reordenação persistida localmente.
- **Quick-Peek Overlay (O "Pulo do Gato"):** Um clique com o botão direito nos itens do menu principal sobrepõe a tela alvo em um modal "glassmorphism" de 94vw/94vh. Isso permite visualizar e checar configurações em outra seção (ex: Políticas RLS) sem perder o editor de código que está aberto no fundo (ex: RPC/Edge).
- **Gestão de Abas Inteligente:** Telas pesadas utilizam abas inferiores resizáveis (Drag-to-Resize Bottom Panels) para Logs/Avisos em vez de scroll vertical forçado.

### 1.3 Internacionalização Soberana e Lean Context
- **Contextual i18n Loading:** O frontend não carrega um "dicionário gigante". Ele solicita os termos específicos da página via API no diretório `/languages`.
- **Zero Cloud Translation:** A interface é 100% nativa. Se um idioma não foi escolhido no `install.sh`, ele não existe no servidor, reduzindo a superfície de ataque e o footprint de armazenamento.

---

## 2. Mapa Estrutural dos Módulos Core

### 2.1 Dashboard Central (Overview Manager)
- **Métricas e Logs Unificados:** Agrega o estado do "Tenant" (Projeto).
- **Indicadores de Deploy:** Trabalha em uníssono com o `DeployWizard` para promover artefatos do Draft para o Live com relatórios de impacto de migração de schema (DDL diffs).

### 2.2 Database Explorer & Modeler (A Ferramenta de Poder)
O coração do BaaS. Muito mais do que um visualizador de tabelas.
- **Table Impact & Column Impact Scanners:** Diferencial arquitetônico maciço. Antes de alterar o tipo de uma coluna ou renomear algo, o "Cascata Dependency Scanner" é acionado. Se o usuário renomear "users" para "clientes", todos os RPCs estritos serão detectados, e um Modal de "Cascade Overwrite" é gerado.
- **Injeção do Cadeado Universal (Column Security):** Controle de RLS e ofuscação (`Hide`, `Blur`, `Mask`, `Encrypt` via Vault) gerido diretamente na malha visual das tabelas.
- **Table Creator Drawer:** Ao criar ou importar tabelas de CSVs/JSONs massivos (Grip & Drop Global), a engine infere automaticamente os tipos primitivos e os apresenta em um *Slide Drawer* na direita, liberando o espaço principal.

### 2.3 RPC, Logic & Edge Manager (Automação Integrada)
Unificamos `RPC` (PosgreSQL Functions), `Triggers`, `Cron Jobs` e `Edge Functions` (V8/Isolates) em um ambiente singular tipo VSCode.
- **Árvore Hierárquica Flexível:** Pastas (`Folders`) criadas de forma visual, agrupando scripts (Drag & Drop com controle de *Ancestors* genéticos para impedir loops infinitos no drop).
- **Resizing IDE + Logs Inferiores:** Como em uma Engine de Games ou IDE moderna, o topo é a edição de código sujo, a parte inferior (`bottomPanelHeight` gerido no localStorage) concentra "Results", "Execution Logs", "Test Params" (Formulário Auto-gerado para teste de funções).
- **Conflito Management:** Detecção prévia de *Overload* de Funções (Mesmo nome, argumentos diferentes). Um Painel de "Diff" para resolver conflitos antes de forçar o Drop Cascade.

### 2.4 Gateway de Autenticação (Auth Config)
- Modelagem de *providers* OAuth, configurações de JWT, duração de Sessão e Magic Links em uma única tela fluida.
- Gerenciamento atômico de contas em formato *DataBrowser* filtrado via RLS de contexto.

### 2.5 Gerente de RLS e Controle de Acesso (Zero-Trust Design)
- **RLS Designer Gráfico:** Um canvas para construir e simular Row Level Security sem exigir experiência prévia pesada em SQL `USING` e `WITH CHECK`. Visualização de quem "Lê", "Insere", "Altera". 

### 2.6 Sistema de Eventos e Push (Push Hub)
- Foco em `Webhooks` diretos atrelados a tabelas (Mutation Observers em tempo real com Batching).
- Envio de SMS/Push massivos consolidados (Manejado via API subjacente com "Test Params" instantâneos em sidebars).

### 2.7 Storage Filesystem (Native Explorer)
- Uma árvore hierárquica baseada no bucket e objeto com preview "Lazy Loaded" de mídias. Editor de RLS de balde (quem pode ler tal subdiretório).

### 2.8 Backup e DR Automático (Templates .caf)
- Empacotador (`CascataArchiveFormat`): Geração automatizada de `snapshots` de infraestrutura e dados via interface, permitindo a restauração granulada de tabelas perdidas ("Recycle Bin Password Override"). Proteção severa de "Admin Password Verification" antes da "Exclusão Definitiva" ou Recriação a partir do .caf.

---

## 3. Pilares Visuais e "Quality of Life"

Para atingir a sensação de ferramenta *High-End* exigida:
1. **Atalhos Embutidos:** Pressionar `Ctrl+S` no RPC Manager não abre o menu "Salvar" do Browser, mas invoca a detecção de Diff, checa sintaxe via Postgres Error Translator e consolida a Query nativa. `Esc` possui um despachante global (Global Esc Handler) que desmonta modais pela ordem Z-Index perfeita (Popups de Deleção -> Options -> Drawers -> Quick Peek).
2. **Postgres Error Translator:** Erros ruidosos ("42P01", "22P02", "23505") não batem crus no front-end. O utilitário intercepta os erros PG nativos e os torna dicas de UI amigáveis vermelhas ou amarelas (Toast sem encobrir texto vital).
3. **Gerador de cURL Híbrido:** O atalho de Cópia Inteligente muda dependendo se sua função é "Edge" ou "RPC", adequando Headers (`apikey: anon_key`) via projeto extraído das chaves "Vault". Tudo ao alcance de um clique para o desenvolvedor compartilhar.

## 4. Próxima Fase

A leitura e refatoração da estrutura nos prova que o Cascata v1.0 já não é um mero CRUD. É uma plataforma que exige **react-virtualized** ou **ag-grid** no *DatabaseExplorer* pela carga de memória, **monaco-editor** profundo nos ambientes de *Logic*, e transições fluídas (Framer Motion) de *Drawers*. A arquitetura delineada reflete perfeitamente a profundidade original, agora estruturada ponta-a-ponta para os passos definitivos no frontend.
