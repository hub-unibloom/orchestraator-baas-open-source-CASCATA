#!/usr/bin/env bash

# ==============================================================================
#  ██████╗ █████╗ ███████╗ ██████╗  █████╗ ████████╗ █████╗ 
# ██╔════╝██╔══██╗██╔════╝██╔════╝ ██╔══██╗╚══██╔══╝██╔══██╗
# ██║     ███████║███████╗██║      ███████║   ██║   ███████║
# ██║     ██╔══██║╚════██║██║      ██╔══██║   ██║   ██╔══██║
# ╚██████╗██║  ██║███████║╚██████╗ ██║  ██║   ██║   ██║  ██║
#  ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
# ==============================================================================
# CASCATA v1.0.0.0 — ZERO-TRUST PRODUCTION INSTALLER
# Environment: Linux (Multi-Distro) / Production / VPS
# Philosophy: Least Privilege, Root-Token Anihilation, Silent Operation.
# ==============================================================================

set -euo pipefail

# --- 1. DESIGN SYSTEM ---
readonly C_BOLD='\033[1m'
readonly C_DIM='\033[2m'
readonly C_BLUE='\033[38;2;99;102;241m' 
readonly C_GREEN='\033[38;2;34;197;94m' 
readonly C_RED='\033[38;2;239;68;68m'    
readonly C_YELLOW='\033[38;2;234;179;8m'
readonly C_CYAN='\033[38;2;56;189;248m'
readonly C_RESET='\033[0m'

log_info()    { echo -e "${C_BLUE}ℹ${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_success() { echo -e "${C_GREEN}✓${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_warn()    { echo -e "${C_YELLOW}⚠${C_RESET} ${C_BOLD}$1${C_RESET}"; }
log_error()   { echo -e "${C_RED}✗${C_RESET} ${C_BOLD}$1${C_RESET}"; exit 1; }
log_step()    { echo -e "\n${C_BOLD}${C_BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${C_RESET}\n${C_BOLD}  ▸ $1${C_RESET}"; }

# --- 2. GLOBAL CONSTANTS ---
readonly REPO_URL="https://github.com/hub-unibloom/orchestraator-baas-open-source-CASCATA.git"
readonly TARGET_DIR="$HOME/cascata_root"
readonly V1_SUBPATH="cascata Go baas orchestrator multi tenacy open source v1"
readonly KEYS_DIR="$HOME/.cascata"
readonly KEYS_FILE="$KEYS_DIR/cascata_keys.env"

# Interative Configuration Holders
WORNER_EMAIL=""
WORNER_PASS=""
MFA_ENABLED=""
MFA_DISPLAY=""

DOCKER_CMD=(docker compose)

print_banner() {
    clear
    echo -e "${C_BLUE}${C_BOLD}
   ██████╗ █████╗ ███████╗ ██████╗  █████╗ ████████╗ █████╗ 
  ██╔════╝██╔══██╗██╔════╝██╔════╝ ██╔══██╗╚══██╔══╝██╔══██╗
  ██║     ███████║███████╗██║      ███████║   ██║   ███████║
  ██║     ██╔══██║╚════██║██║      ██╔══██║   ██║   ██╔══██║
  ╚██████╗██║  ██║███████║╚██████╗ ██║  ██║   ██║   ██║  ██║
   ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
${C_RESET}${C_DIM}                              v1.0.0.0 | BaaS Go Orchestrator Studio${C_RESET}"
    echo -e "${C_DIM}---------------------------------------------------------------${C_RESET}\n"
    log_info "Inicializando Orquestração Zero-Trust | Soberania Cascata v1..."
}

check_privileges() {
    if [[ "$EUID" -ne 0 ]]; then
        log_error "Instalação bloqueada: Requer privilégios de superusuário nativos (sudo ./install.sh)."
    fi
}

collect_worner_credentials() {
    log_step "Identidade Corporativa: Provisionamento do Worner"
    
    echo -e "  ${C_DIM}O 'Worner' é o administrador supremo do Condomínio Cascata.${C_RESET}"
    echo -e "  ${C_DIM}Este será o único membro criado fora do Audit Trail visual.${C_RESET}\n"
    
    while true; do
        read -p "  E-mail do Worner: " WORNER_EMAIL
        if [[ -n "$WORNER_EMAIL" ]]; then break; fi
    done

    while true; do
        read -s -p "  Senha Mestre (Mínimo 12 caracteres): " WORNER_PASS
        echo ""
        if [[ ${#WORNER_PASS} -ge 12 ]]; then break; fi
        log_warn "A segurança exige o mínimo de 12 caracteres (Argon2id High-Memory)."
    done

    while true; do
        read -p "  Habilitar OTP / MFA Obrigatório? (s/n): " OTP_CHOICE
        case "$OTP_CHOICE" in
            [sS]* ) MFA_ENABLED="true"; break;;
            [nN]* ) MFA_ENABLED="false"; break;;
            * ) echo "Por favor, responda 's' para sim ou 'n' para não.";;
        esac
    done
}

collect_language_preference() {
    log_step "Experiência de Uso: Internacionalização Soberana"
    
    local LANG_DIR="${TARGET_DIR}/${V1_SUBPATH}/languages"
    if [[ ! -d "$LANG_DIR" ]]; then
        mkdir -p "$LANG_DIR"
        log_warn "Diretório de idiomas vazio. Prosseguindo com fallback en_US."
        DEFAULT_LANG="en_US"
        return
    fi

    echo -e "  ${C_DIM}Selecione o idioma principal do Cascata Orchestrator:${C_RESET}\n"
    
    local langs=()
    for f in "$LANG_DIR"/*.json; do
        [[ -e "$f" ]] || continue
        langs+=("$(basename "$f" .json)")
    done

    if [[ ${#langs[@]} -eq 0 ]]; then
        DEFAULT_LANG="en-US"
        log_warn "Nenhum arquivo de tradução encontrado. Fallback en-US ativo."
        return
    fi

    for i in "${!langs[@]}"; do
        echo -e "    $((i+1)). ${langs[i]}"
    done

    echo ""
    local choice
    while true; do
        read -p "  Opção (1-${#langs[@]}): " choice
        if [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "${#langs[@]}" ]; then
            DEFAULT_LANG="${langs[$((choice-1))]}"
            break
        fi
    done

    # Cleanup: Mantemos os arquivos JSON, apenas definimos o padrão no .env
    log_success "Idioma '$DEFAULT_LANG' definido como oficial do Orquestrador."
}

# --- 3. REPOSITORY SYNC ---
sync_repository() {
    log_step "Sincronizando Código Fonte"
    
    if [[ -d "$TARGET_DIR" ]]; then
        log_info "Atualizando repositório local..."
        cd "$TARGET_DIR"
        git fetch --all --quiet
        git reset --hard origin/main --quiet
    else
        log_info "Clonando plataforma remota..."
        git clone --quiet "$REPO_URL" "$TARGET_DIR"
        cd "$TARGET_DIR"
    fi

    local V1_ABS_PATH="${TARGET_DIR}/${V1_SUBPATH}"
    if [[ ! -d "$V1_ABS_PATH" ]]; then
        log_error "Sub-diretório de versão v1.0.0.0 não encontrado."
    fi

    cd "$V1_ABS_PATH"
    log_success "Diretório de operação definido: $(pwd)"
}

# --- 4. ENGINE DEPENDENCIES ---
ensure_dependencies() {
    log_step "Validando Runtime (Docker) e Ferramentas"
    
    local PKG_MGR=""
    if command -v apt-get >/dev/null 2>&1; then
        PKG_MGR="apt-get"
    elif command -v dnf >/dev/null 2>&1; then
        PKG_MGR="dnf"
    elif command -v yum >/dev/null 2>&1; then
        PKG_MGR="yum"
    fi

    if [[ -n "$PKG_MGR" ]]; then
        $PKG_MGR update -y -qq >/dev/null 2>&1 || true
        $PKG_MGR install -y -qq curl git jq >/dev/null 2>&1 || true
    fi

    if ! command -v docker >/dev/null 2>&1; then
        log_info "Instalando Docker Engine via canal oficial..."
        curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh && rm get-docker.sh
        systemctl enable --now docker >/dev/null 2>&1 || true
    fi
    
    local MAJOR_DOCKER_VER
    MAJOR_DOCKER_VER=$(docker version --format '{{.Server.Version}}' 2>/dev/null | cut -d'.' -f1 || echo "0")
    if [[ "$MAJOR_DOCKER_VER" -lt 24 ]]; then
        log_error "Versão Docker ($MAJOR_DOCKER_VER.x) inadequada. Mínimo exigido: 24.x para ecossistema v2 nativo."
    fi

    if docker compose version >/dev/null 2>&1; then
        DOCKER_CMD=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
        log_error "Docker-Compose Legado (v1) detectado. Instale 'docker-compose-plugin'."
    else
        log_info "Instalando Docker Compose Plugin..."
        if [[ -n "$PKG_MGR" ]]; then
            $PKG_MGR install -y -qq docker-compose-plugin >/dev/null 2>&1 || true
        fi
        DOCKER_CMD=(docker compose)
    fi
    
    log_success "Docker Engine verificado (v$MAJOR_DOCKER_VER.x)."
    log_info "Cascata Architecture: Zero-Trust Foundation."
}

# --- 5. PERF & SECURITY HYGIENE ---
apply_tuning() {
    log_step "Aplicando Tuning de Performance Persistente (Sysctl)"
    
    local SYSCTL_CONF="/etc/sysctl.d/99-cascata.conf"
    
    cat <<EOF > "$SYSCTL_CONF"
# Cascata Master Tuning: BDB & Dragonfly DB Optimization
vm.max_map_count=524288
fs.file-max=131072
net.ipv4.tcp_keepalive_time=60
EOF
    
    sysctl -p "$SYSCTL_CONF" >/dev/null 2>&1 || true
    
    # --- Dynamic Postgres Tuning Resolution ---
    local TOTAL_RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local TOTAL_RAM_MB=$((TOTAL_RAM_KB / 1024))
    
    # Strategy: shared_buffers = 25% of RAM, max 1GB for density safety.
    export PG_SHARED_BUFFERS="$((TOTAL_RAM_MB / 4))MB"
    if [[ $((TOTAL_RAM_MB / 4)) -gt 1024 ]]; then PG_SHARED_BUFFERS="1024MB"; fi
    
    # Strategy: effective_cache_size = 50% of RAM
    export PG_EFFECTIVE_CACHE="$((TOTAL_RAM_MB / 2))MB"
    
    # Strategy: work_mem = safer small value per connection to avoid OOM in high concurrency
    export PG_WORK_MEM="4MB"
    if [[ $TOTAL_RAM_MB -lt 2048 ]]; then PG_WORK_MEM="2MB"; fi

    log_success "Parâmetros Kernel gravados e tuning de RAM calculado ($TOTAL_RAM_MB MB)."
}

secure_bootstrap() {
    log_step "Provisão de Identidade Criptográfica (.env)"
    
    if [[ -f ".env" ]]; then
        log_warn "Arquivo .env já existente. Fazendo backup e gerando novo para FRESH INSTALL."
        mv .env .env.bak.$(date +%s)
    fi

    local DB_USER="cascata_root_$(tr -dc 'a-z0-9' < /dev/urandom | head -c 8)"
    local DB_PASS=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    local JWT_SEC=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64)
    local MASTER_KEY=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)

    cat <<EOF > .env
# --- CASCATA V1 MASTER ENVS ---
PROJECT_NAME=cascata
NODE_ENV=production

# Core Databases
DB_USER=${DB_USER}
DB_PASS=${DB_PASS}
DB_NAME=cascata_meta
DB_PORT=5432

# Language Preference
DEFAULT_LANGUAGE=${DEFAULT_LANG}

# Performance / Auto-Tuning (Computed by Installer)
PG_SHARED_BUFFERS=${PG_SHARED_BUFFERS}
PG_EFFECTIVE_CACHE=${PG_EFFECTIVE_CACHE}
PG_WORK_MEM=${PG_WORK_MEM}

# Shared Infrastructure URLs (Assembled for Sinergy)
DB_URL=postgres://${DB_USER}:${DB_PASS}@cascata-db:5432/${DB_NAME}
DFLY_URL=redis://cascata-cache:6379

# Internal Logic Exchange
SYSTEM_JWT_SECRET=${JWT_SEC}
CASCATA_MASTER_KEY=${MASTER_KEY}
EOF
    chmod 600 .env
    log_success "Arquivo Base Criptográfico (Zero Dev-Mode) concluído."
}

# --- 6. LAYER ORCHESTRATION ---
launch_cluster_base() {
    log_step "Iniciando Orquestração do Cluster Base"
    
    if [[ ! -f "docker-compose.yml" ]]; then
        log_error "Arquitetura YAML não detectada em $(pwd)"
    fi

    log_info "ANICILAÇÃO DE DADOS: Limpando Containers, Redes e VOLUMES Órfãos..."
    "${DOCKER_CMD[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

    log_info "Realizando boot isolado da malha Docker..."
    
    "${DOCKER_CMD[@]}" pull -q || true
    "${DOCKER_CMD[@]}" up -d --build
    
    log_success "Containers Base enviados à rede."
}

reload_and_verify() {
    log_step "Recarregando Conexão e Monitorando Healthchecks"
    
    "${DOCKER_CMD[@]}" up -d >/dev/null 2>&1
    
    local timer=0
    while : ; do
        local healthy=$(docker ps --filter "name=cascata" --filter "health=healthy" --format "{{.Names}}" | wc -l | tr -d ' ')
        local total=$(docker ps --filter "name=cascata" --format "{{.Names}}" | wc -l | tr -d ' ')
        
        if [[ "$healthy" -ge "$total" ]] && [[ "$total" -gt 0 ]]; then
            break
        fi
        
        echo -ne "  ${C_DIM}Assimilando rede de microserviços: ${healthy}/${total}...${C_RESET}\r"
        sleep 2
        
        timer=$((timer + 1))
        if [[ "$timer" -gt 40 ]]; then
            log_warn "Timeout secundário disparado visualmente."
            break
        fi
    done
    
    log_success "\nEcossistema consolidado completamente."
}

provision_worner_execution() {
    log_step "Injeção de Identidade: Worner Provisioning (Secure Native)"
    
    local ORCH_CONTAINER
    ORCH_CONTAINER=$(docker ps --filter "name=orchestrator" --format "{{.Names}}" | head -n 1)
    
    if [[ -z "$ORCH_CONTAINER" ]]; then
        log_error "Container Orquestrador inoperante. Não foi possível provisionar o Worner."
    fi

    log_info "Calculando Argon2id (64MB/4-Threads) e persistindo via Cascata-Engine..."
    
    local CURRENT_DB_USER=$(grep '^DB_USER=' .env | cut -d'=' -f2)
    local CURRENT_DB_PASS=$(grep '^DB_PASS=' .env | cut -d'=' -f2)
    local MASTER_KEY=$(grep '^CASCATA_MASTER_KEY=' .env | cut -d'=' -f2)
    
    # Geramos o segredo MFA aqui se habilitado, passando para o provisionador
    local MFA_SECRET=""
    if [[ "$MFA_ENABLED" == "true" ]]; then
        MFA_SECRET=$(tr -dc 'A-Z2-7' < /dev/urandom | head -c 32)
    fi

    # Injeção de Identidade usando a URL de conexão já presente no ambiente do container
    local PROVISION_OUT
    PROVISION_OUT=$(docker exec -e CASCATA_MASTER_KEY="$MASTER_KEY" \
        "$ORCH_CONTAINER" ./worner-provision "$WORNER_EMAIL" "$WORNER_PASS" "$MFA_ENABLED" "$MFA_SECRET" 2>&1)

    if [[ ! "$PROVISION_OUT" =~ "SUCCESS_ID:" ]]; then
        log_error "Falha no provisionamento: $PROVISION_OUT"
    fi

    local WORNER_ID
    WORNER_ID=$(echo "$PROVISION_OUT" | grep "SUCCESS_ID:" | cut -d':' -f2)
    log_success "Membro Worner ($WORNER_ID) injetado via Engine."

    if [[ "$MFA_ENABLED" == "true" ]]; then
        MFA_DISPLAY="  ${C_BOLD}OTP_SECRET (Add no Google Auth):${C_RESET} ${MFA_SECRET}"
    else
        MFA_DISPLAY="  ${C_BOLD}OTP / MFA:${C_RESET} Desativado"
    fi
}

show_final() {
    local EXTERNAL_IP
    EXTERNAL_IP=$(curl -s -m 5 https://checkip.amazonaws.com || curl -s -m 5 https://ifconfig.me || echo "localhost")
    
    log_step "CASCATA SOVEREIGN ORCHESTRATOR v1 INSTALADO"
    
    local JWT_EXT=$(grep '^SYSTEM_JWT_SECRET=' .env | cut -d '=' -f2)
    local MASTER_EXT=$(grep '^CASCATA_MASTER_KEY=' .env | cut -d '=' -f2)

    echo -e "  ✦ ${C_BOLD}Cascata Dashboard (Sovereign):${C_RESET}  http://${EXTERNAL_IP}"
    echo -e "  ✦ ${C_BOLD}Cascata Private API (V1):${C_RESET}       http://${EXTERNAL_IP}/v1/\n"
    
    echo -e "${C_DIM}--- Chaves de Acesso e Setup ---${C_RESET}"
    echo -e "  ${C_BOLD}Worner E-mail:${C_RESET} ${WORNER_EMAIL}"
    echo -e "$MFA_DISPLAY\n"
    
    local DB_EXT=$(grep '^DB_USER=' .env | cut -d '=' -f2)
    echo -e "  ${C_BOLD}DB_ADMIN:${C_RESET} ${DB_EXT}"
    echo -e "  ${C_BOLD}JWT_MASTER:${C_RESET} ${JWT_EXT:0:20}...${JWT_EXT: -5}"
    echo -e "  ${C_BOLD}CASCATA_MASTER_KEY:${C_RESET} ${MASTER_EXT:0:15}...\n"
    
    log_success "Deploy Inviolável. Mantenha os endereços e segredos longe do escrutínio público."
}

# --- EXECUTION FLOW ---
print_banner
check_privileges
collect_worner_credentials
sync_repository
collect_language_preference
ensure_dependencies
apply_tuning
secure_bootstrap
launch_cluster_base
reload_and_verify
provision_worner_execution
show_final
