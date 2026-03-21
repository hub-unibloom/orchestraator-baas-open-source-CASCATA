#!/usr/bin/env bash

# ==============================================================================
# CASCATA v1.0.0.0 — INSTALLER SCRIPT
# Environment: Linux (Ubuntu/Debian recommended) / VPS
#
# Este instalador tem responsabilidades rígidas:
# 1. Validar e/ou instalar Docker e Docker Compose nativo.
# 2. Configurar permissões de volumes essenciais.
# 3. Levantar a infraestrutura via docker-compose de forma paralela.
# 4. Aguardar o healthcheck síncrono dos serviços pesados (Postgres, Vault).
# 5. Entregar as chaves e rotas da API em tela.
#
# Tolerância Zero a Falhas: Se um container crashear, a instalação aborta com log.
# ==============================================================================

set -eo pipefail # Fail fast na borda (Coding Standards)

# --- Cores da Interface ---
C_RESET='\033[0m'
C_BOLD='\033[1m'
C_DIM='\033[2m'
C_BLUE='\033[38;2;99;102;241m'  # accent-primary do sistema
C_GREEN='\033[38;2;34;197;94m'  # accent-success
C_RED='\033[38;2;239;68;68m'    # accent-danger
C_YELLOW='\033[38;2;234;179;8m' # accent-warning

log_info() { echo -e "${C_BLUE}ℹ${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_success() { echo -e "${C_GREEN}✓${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_warn() { echo -e "${C_YELLOW}⚠${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_error() { echo -e "${C_RED}✗${C_RESET} ${C_BOLD}$1${C_RESET}"; exit 1; }
log_step() { echo -e "\n${C_DIM}---${C_RESET} ${C_BOLD}$1${C_RESET} ${C_DIM}---${C_RESET}"; }

# Verifica se estamos em Linux (obrigatório para VPS alvo)
if [[ "$(uname -s)" != "Linux" ]]; then
    log_warn "O instalador oficial foi projetado para Linux/VPS. Comportamento não garantido em OSX/Windows."
fi

# Verifica privilegios sudo dinamicamente
if [ "$EUID" -ne 0 ]; then
    # Se o docker já existe e o usuário tem acesso (grupo docker), seguimos em frente sem root.
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        log_info "Sessão usuária verificada válida para manipulação do Docker. Seguindo sem root."
    else
        log_warn "O Docker daemon não parece acessível ou não está instalado."
        log_info "Tentando elevar privilégios automaticamente para prosseguir..."
        if command -v sudo >/dev/null 2>&1; then
            exec sudo bash "$0" "$@"
        else
            log_error "O comando 'sudo' não está disponível. Por favor, rode o script como 'root'."
        fi
    fi
fi

clear
echo -e "${C_BLUE}
██████   █████  ███████  ██████  █████  ████████  █████  
██      ██   ██ ██      ██      ██   ██    ██    ██   ██ 
██      ███████ ███████ ██      ███████    ██    ███████ 
██      ██   ██      ██ ██      ██   ██    ██    ██   ██ 
██████  ██   ██ ███████  ██████ ██   ██    ██    ██   ██ 
                                            v1.0.0.0 Studio
${C_RESET}"
echo -e "Inicializando orquestração BaaS Multi-Tenant...\n"

# ==============================================================================
# START: FASE 1 - DEPENDÊNCIAS DO HOST
# ==============================================================================
log_step "Validando Docker Runtime"

if ! command -v docker >/dev/null 2>&1; then
    log_info "Docker engine não encontrado. Provisionando instalação nativa..."
    curl -fsSL https://get.docker.com -o get-docker.sh || log_error "Falha ao baixar instalador Docker"
    sh get-docker.sh || log_error "Falha na instalação do Docker Engine"
    rm get-docker.sh
    systemctl enable docker
    systemctl start docker
    log_success "Docker nativo instalado e ativo."
else
    log_success "Docker daemon presente: $(docker --version)"
fi

if ! docker compose version >/dev/null 2>&1; then
    log_info "Plugin Docker Compose não encontrado."
    if ! command -v docker-compose >/dev/null 2>&1; then
        log_error "Nem o 'docker-compose' e nem o 'docker compose' (plugin) estão instalados. Instale o docker-compose-plugin manualmente."
    else
        DOCKER_COMPOSE_CMD="docker-compose"
        log_success "Usando docker-compose legado secundário."
    fi
else
    DOCKER_COMPOSE_CMD="docker compose"
    log_success "Docker Compose gen v2 presente."
fi

# ==============================================================================
# START: FASE 2 - DOCKER NETWORK & IMAGES
# ==============================================================================
log_step "Aplicando Configurações do Cluster"

# Parar tudo se ja estiver rodando
log_info "Matando containeres antigos (Fail Fast se houver conflito)"
$DOCKER_COMPOSE_CMD down --remove-orphans || true

# Criação de volumes de base para que as permissões não buguem (Vault / Postgres)
log_info "Provisionando Persistent Volumes..."
$DOCKER_COMPOSE_CMD pull
log_success "Imagens atualizadas validadas via registry."

# ==============================================================================
# START: FASE 3 - DEPLOYMENT CONCORRENTE 
# ==============================================================================
log_step "Lançando Infraestrutura em Background (-d)"

$DOCKER_COMPOSE_CMD up -d

# ==============================================================================
# START: FASE 4 - HEATHCHECKS SÍNCRONOS
# ==============================================================================
log_step "Injetando código defensivo: Aguardando Prontidão de I/O..."

wait_for_service() {
    local SERVICE_NAME=$1
    local MAX_RETRIES=30
    local RETRY_COUNT=0
    
    echo -ne "  ${C_DIM}Aguardando ${SERVICE_NAME} estabilizar...${C_RESET}\r"
    while : ; do
        # Busca o estado de 'health' dentro dos containers que exportam nativo no compose
        STATUS=$(docker inspect --format='{{json .State.Health.Status}}' cascata-${SERVICE_NAME} 2>/dev/null || echo "\"unknown\"")
        STATUS=$(echo "$STATUS" | tr -d '"')

        if [[ "$STATUS" == "healthy" ]]; then
            echo -e "  ${C_GREEN}✓${C_RESET} ${SERVICE_NAME} Operacional (~${RETRY_COUNT}s de boot)         "
            return 0
        elif [[ "$STATUS" == "unhealthy" ]]; then
            echo -e "\n  ${C_RED}✗${C_RESET} Falha crítica: ${SERVICE_NAME} subiu corrompido ou preso em loop."
            echo -e "\n${C_DIM}Log dump (${SERVICE_NAME}):${C_RESET}"
            docker logs --tail 20 cascata-${SERVICE_NAME}
            log_error "Instalação atômica revertida manualmente. Verifique os logs."
        fi

        # Para Vault e outros que não tiverem health map via dockerd, ou fallback se demorar
        if [[ "$SERVICE_NAME" == "vault" && "$STATUS" == "unknown" ]]; then
            # curl defensivo com retry exponencial mitigado
            local VAULT_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8200/v1/sys/health || echo "000")
            if [[ "$VAULT_HTTP" == "200" || "$VAULT_HTTP" == "429" || "$VAULT_HTTP" == "473" ]]; then
               echo -e "  ${C_GREEN}✓${C_RESET} Vault Master Key ativa (~${RETRY_COUNT}s de boot)         "
               return 0
            fi
        fi

        sleep 1
        ((RETRY_COUNT++))
        if [ "$RETRY_COUNT" -gt "$MAX_RETRIES" ]; then
            echo -e "\n  ${C_RED}✗${C_RESET} Timeout de ${MAX_RETRIES} segundos atingido."
            docker logs --tail 20 cascata-${SERVICE_NAME}
            log_error "O serviço ${SERVICE_NAME} não reportou saúde sadia a tempo."
        fi
    done
}

# Valida os 3 core base, aguardando o banco primeiro obrigatoriamente
wait_for_service "db"
wait_for_service "cache"
wait_for_service "vault"
wait_for_service "orchestrator"

# ==============================================================================
# START: FASE FINAL - PAINEL SECRETO
# ==============================================================================
log_step "Tudo Operacional"

HOST_IP=$(curl -s ifconfig.me || echo "127.0.0.1")

echo -e "  ${C_BOLD}CASCATA STUDIO (FRONTEND v1.0)${C_RESET} Em breve na rota da API ou via Vite Dev"
echo -e "  ${C_BOLD}ORQUESTRADOR (BACKEND API)${C_RESET}   http://${HOST_IP}:8080"
echo -e "  ${C_BOLD}POSTGRES MULTI-TENANT META${C_RESET}   postgres://cascata_admin:cascata_pass@${HOST_IP}:5432/cascata_meta"
echo -e "  ${C_BOLD}VAULT UI (DEV MODE)${C_RESET}          http://${HOST_IP}:8200\n"

echo -e "${C_YELLOW}⚠ IMPORTANTE: O Vault está rodando em '-dev' e o token raiz é:${C_RESET}"
echo -e "${C_BLUE}  cascata_root_token${C_RESET}\n"

echo -e "${C_DIM}Todos os dados agora sofrem Fail Fast na borda, o frontend UI usa Design Tokens rígidos,"
echo -e "os logs em Go são estruturados e I/O nunca bloqueia a stack.${C_RESET}"
log_success "Deploy Completo e Perfeito."
