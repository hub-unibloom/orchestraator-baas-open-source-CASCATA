#!/usr/bin/env bash

# ==============================================================================
# CASCATA v1.0.0.0 — REAL PLUG & PLAY INSTALLER SCRIPT
# Environment: Linux (Ubuntu/Debian recommended) / VPS
#
# Propósito Deste Instalador (Plug & Play Real):
# 1. Auto-elevar privilégios caso baixado e rodado via usuário normal.
# 2. Instalar dependências de sistema host (Git, curl, Docker, Compose).
# 3. Clonar o repositório completo (pois o docker-compose exige a pasta /scripts e etc).
# 4. Garantir que os serviços essenciais inicializem e estejam hard-tested.
# ==============================================================================

set -eo pipefail # Fail fast

# --- Cores da Interface ---
C_RESET='\033[0m'
C_BOLD='\033[1m'
C_DIM='\033[2m'
C_BLUE='\033[38;2;99;102;241m' 
C_GREEN='\033[38;2;34;197;94m' 
C_RED='\033[38;2;239;68;68m'    
C_YELLOW='\033[38;2;234;179;8m'

log_info() { echo -e "${C_BLUE}ℹ${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_success() { echo -e "${C_GREEN}✓${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_warn() { echo -e "${C_YELLOW}⚠${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_error() { echo -e "${C_RED}✗${C_RESET} ${C_BOLD}$1${C_RESET}"; exit 1; }
log_step() { echo -e "\n${C_DIM}---${C_RESET} ${C_BOLD}$1${C_RESET} ${C_DIM}---${C_RESET}"; }

# Configurações do Repositório Alvo
REPO_URL="https://github.com/hub-unibloom/orchestraator-baas-open-source-CASCATA.git"
INSTALL_DIR="/opt/cascata-v1"
PROJECT_FOLDER="cascata Go baas orchestrator multi tenacy open source v1"

clear
echo -e "${C_BLUE}
██████   █████  ███████  ██████  █████  ████████  █████  
██      ██   ██ ██      ██      ██   ██    ██    ██   ██ 
██      ███████ ███████ ██      ███████    ██    ███████ 
██      ██   ██      ██ ██      ██   ██    ██    ██   ██ 
██████  ██   ██ ███████  ██████ ██   ██    ██    ██   ██ 
                                            v1.0.0.0 Studio
${C_RESET}"
echo -e "Inicializando instalador Plug & Play Cascata...\n"

# ==============================================================================
# 1. VERIFICAÇÃO DE ROOT AUTOMÁTICA
# ==============================================================================
if [ "$EUID" -ne 0 ]; then
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        log_info "Sessão atual possui acesso ao Docker. Prosseguindo."
    else
        log_warn "O Docker não está acessível ou necessitamos de privilégios para instalar pacotes."
        log_info "Tentando elevar com sudo automaticamente..."
        if command -v sudo >/dev/null 2>&1; then
            exec sudo bash "$0" "$@"
        else
            log_error "Comando 'sudo' ausente. Rode 'sudo ./install.sh' ou logue como root."
        fi
    fi
fi

# ==============================================================================
# 2. INSTALAÇÃO DE PACOTES BÁSICOS DO HOST (Git, curl)
# ==============================================================================
log_step "Verificando pacotes vitais (Git/Curl)"
if ! command -v git >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
    log_info "Git ou Curl ausentes. Instalando via apt-get..."
    apt-get update -y >/dev/null 2>&1 || true
    apt-get install -y git curl >/dev/null 2>&1 || log_error "Falha ao instalar Git/Curl nativo."
fi
log_success "Git e Curl operacionais."

# ==============================================================================
# 3. INSTALAÇÃO DO DOCKER
# ==============================================================================
log_step "Validando Docker Runtime"

if ! command -v docker >/dev/null 2>&1; then
    log_info "Docker engine não encontrado na VPS. Provisionando via get.docker.com..."
    curl -fsSL https://get.docker.com -o get-docker.sh || log_error "Falha ao baixar script oficial do Docker."
    sh get-docker.sh >/dev/null 2>&1 || log_error "Falha na instalação do Docker Engine."
    rm get-docker.sh
    systemctl enable docker >/dev/null 2>&1 || true
    systemctl start docker >/dev/null 2>&1 || true
    log_success "Docker nativo instalado."
else
    log_success "Docker daemon presente: $(docker --version)"
fi

if ! docker compose version >/dev/null 2>&1; then
    if ! command -v docker-compose >/dev/null 2>&1; then
        log_info "Docker Compose não detectado. Instalando plugin nativo..."
        apt-get install -y docker-compose-plugin >/dev/null 2>&1 || log_error "Falha instalando Compose Plugin."
        DOCKER_COMPOSE_CMD="docker compose"
    else
        DOCKER_COMPOSE_CMD="docker-compose"
    fi
else
    DOCKER_COMPOSE_CMD="docker compose"
fi

# ==============================================================================
# 4. CLONE DO REPOSITÓRIO FÍSICO (A MÁGICA DO PLUG & PLAY)
# ==============================================================================
log_step "Sincronizando Código Fonte Oficial ($INSTALL_DIR)"

if [ -d "$INSTALL_DIR" ]; then
    log_info "Diretório $INSTALL_DIR já existe. Atualizando via Git pull..."
    cd "$INSTALL_DIR"
    git reset --hard HEAD >/dev/null 2>&1
    git pull origin main >/dev/null 2>&1 || log_warn "Não foi possivel fazer hard pull. Usando a versão local do diretório."
else
    log_info "Clonando repositório para a VPS..."
    git clone "$REPO_URL" "$INSTALL_DIR" >/dev/null 2>&1 || log_error "Falha ao clonar repositório."
fi

# Navega especificamente para a sub-pasta do projeto v1
cd "$INSTALL_DIR/$PROJECT_FOLDER" || log_error "Diretório alvo '$PROJECT_FOLDER' não encontrado dentro do repositório."
log_success "Repositório baixado e posicionado na raiz do Orquestrador v1."

# ==============================================================================
# 5. DEPLOYMENT (DOCKER COMPOSE)
# ==============================================================================
log_step "Aplicando Configurações e Orquestrando Cluster"

log_info "Baixando e alinhando imagens base..."
$DOCKER_COMPOSE_CMD pull -q || true

log_info "Iniciando infraestrutura principal do Cascata (-d)..."
$DOCKER_COMPOSE_CMD up -d

# ==============================================================================
# 6. HEATHCHECKS SÍNCRONOS
# ==============================================================================
log_step "Injetando código defensivo: Aguardando Servidores de Apoio..."

wait_for_service() {
    local SERVICE_NAME=$1
    local MAX_RETRIES=40
    local RETRY_COUNT=0
    
    echo -ne "  ${C_DIM}Aguardando ${SERVICE_NAME} estabilizar...${C_RESET}\r"
    while : ; do
        STATUS=$(docker inspect --format='{{json .State.Health.Status}}' cascata-${SERVICE_NAME} 2>/dev/null || echo "\"unknown\"")
        STATUS=$(echo "$STATUS" | tr -d '"')

        if [[ "$STATUS" == "healthy" ]]; then
            echo -e "  ${C_GREEN}✓${C_RESET} ${SERVICE_NAME} Operacional (~${RETRY_COUNT}s de boot)         "
            return 0
        elif [[ "$STATUS" == "unhealthy" ]]; then
            echo -e "\n  ${C_RED}✗${C_RESET} Falha crítica: ${SERVICE_NAME} quebrou loop de teste."
            docker logs --tail 20 cascata-${SERVICE_NAME}
            log_error "Orquestração falhou. Verifique logs e recursos de Hardware da VPS."
        fi

        # Para serviços que não declaram healthcheck nativo
        if [[ "$SERVICE_NAME" == "vault" && "$STATUS" == "unknown" ]]; then
            local VAULT_HTTP=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8200/v1/sys/health || echo "000")
            if [[ "$VAULT_HTTP" == "200" || "$VAULT_HTTP" == "429" || "$VAULT_HTTP" == "473" ]]; then
               echo -e "  ${C_GREEN}✓${C_RESET} Vault Master Key ativa (~${RETRY_COUNT}s de boot)         "
               return 0
            fi
        fi

        sleep 1
        ((RETRY_COUNT++))
        if [ "$RETRY_COUNT" -gt "$MAX_RETRIES" ]; then
            echo -e "\n  ${C_RED}✗${C_RESET} Timeout atingido para ${SERVICE_NAME}."
            docker logs --tail 20 cascata-${SERVICE_NAME}
            log_error "O serviço ${SERVICE_NAME} não iniciou a tempo."
        fi
    done
}

# Aguardamos os 4 pilares:
wait_for_service "db"
wait_for_service "cache"
wait_for_service "vault"
wait_for_service "orchestrator"

# ==============================================================================
# END: PAINEL SECRETO E SUCESSO
# ==============================================================================
log_step "Plataforma Hospedada Ativa"

HOST_IP=$(curl -s ifconfig.me || echo "127.0.0.1")

echo -e "  ${C_BOLD}CASCATA STUDIO (FRONTEND v1.0)${C_RESET} Acesse de seu computador de Dev ou aguarde a porta Front."
echo -e "  ${C_BOLD}ORQUESTRADOR (BACKEND API)${C_RESET}   http://${HOST_IP}:8080"
echo -e "  ${C_BOLD}POSTGRES MULTI-TENANT META${C_RESET}   postgres://cascata_admin:cascata_pass@${HOST_IP}:5432/cascata_meta"
echo -e "  ${C_BOLD}VAULT UI (DEV MODE)${C_RESET}          http://${HOST_IP}:8200\n"

echo -e "${C_YELLOW}⚠ IMPORTANTE - CREDENCIAIS DEV:${C_RESET}"
echo -e "${C_BLUE}  Vault Token:${C_RESET} cascata_root_token\n"

echo -e "${C_DIM}A pasta oficial do projeto ficou na VPS em: $INSTALL_DIR${C_RESET}"
log_success "Deploy Concluído perfeitamente. Pode fechar o terminal e agir."
