#!/usr/bin/env bash

# ==============================================================================
#  ██████╗ █████╗ ███████╗ ██████╗  █████╗ ████████╗ █████╗ 
# ██╔════╝██╔══██╗██╔════╝██╔════╝ ██╔══██╗╚══██╔══╝██╔══██╗
# ██║     ███████║███████╗██║      ███████║   ██║   ███████║
# ██║     ██╔══██║╚════██║██║      ██╔══██║   ██║   ██╔══██║
# ╚██████╗██║  ██║███████║╚██████╗ ██║  ██║   ██║   ██║  ██║
#  ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
# ==============================================================================
# CASCATA v1.0.0.0 — FULL PLUG & PLAY INSTALLER
# Environment: Linux (Multi-Distro) / Production / VPS
# Solution: Robust Command Dispatching via Bash Arrays.
# ==============================================================================

set -euo pipefail
IFS=$'\n\t'

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
${C_RESET}${C_DIM}                              v1.0.0.0 | Orchestrator Studio${C_RESET}"
    echo -e "${C_DIM}---------------------------------------------------------------${C_RESET}\n"
    log_info "Inicializando orquestração BaaS Multi-Tenant..."
}

check_privileges() {
    if [ "$EUID" -ne 0 ]; then
        log_warn "O instalador necessita de privilégios elevados."
        if command -v sudo >/dev/null 2>&1; then
            log_info "Elevando privilégios automaticamente..."
            exec sudo bash "$0" "$@"
        else
            log_error "Permissão negada. Execute: sudo ./install.sh"
        fi
    fi
}

# --- 3. REPOSITORY SYNC ---
sync_repository() {
    log_step "Sincronizando Código Fonte e Arquivos de Orquestração"
    
    if [ -d "$TARGET_DIR" ]; then
        log_info "Atualizando repositório em $TARGET_DIR..."
        cd "$TARGET_DIR"
        git fetch --all --quiet
        git reset --hard origin/main --quiet
    else
        log_info "Clonando plataforma do repositório remoto..."
        git clone --quiet "$REPO_URL" "$TARGET_DIR"
        cd "$TARGET_DIR"
    fi

    local V1_ABS_PATH="$TARGET_DIR/$V1_SUBPATH"
    if [ ! -d "$V1_ABS_PATH" ]; then
        log_error "Sub-diretório '$V1_SUBPATH' não encontrado. Repositório incompleto."
    fi

    cd "$V1_ABS_PATH"
    log_success "Sincronizado e posicionado na raiz de Orquestração v1."
}

# --- 4. DEPENDENCIES & DISPATCHER ---
ensure_dependencies() {
    log_step "Validando Runtime (Docker) e Ferramentas"
    
    # OS Detection
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        case $ID in
            ubuntu|debian|pop|mint) apt-get update -qq && apt-get install -qq -y curl git jq >/dev/null ;;
            centos|rhel|almalinux|rocky|fedora) dnf install -y -q curl git jq >/dev/null ;;
        esac
    fi

    # Docker Install
    if ! command -v docker >/dev/null 2>&1; then
        log_info "Instalando Docker Engine canal oficial..."
        curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh && rm get-docker.sh
        systemctl enable --now docker >/dev/null 2>&1 || true
    fi
    
    # Docker Compose Detection (Array-Ready)
    if docker compose version >/dev/null 2>&1; then
        DOCKER_CMD=(docker compose)
        log_info "Usando Docker Compose (V2 Plugin)"
    elif command -v docker-compose >/dev/null 2>&1; then
        DOCKER_CMD=(docker-compose)
        log_warn "Usando Docker-Compose (V1 Legado)"
    else
        log_info "Provisionando Docker Compose Plugin (V2)..."
        apt-get install -y docker-compose-plugin >/dev/null 2>&1 || true
        DOCKER_CMD=(docker compose)
    fi
    
    log_success "Docker Engine pronto: $(docker --version)"
}

# --- 5. PERF & SECURITY ---
apply_tuning() {
    log_step "Aplicando Tuning de Performance (Kernel)"
    sysctl -w vm.max_map_count=524288 >/dev/null 2>&1 || true
    sysctl -w fs.file-max=131072 >/dev/null 2>&1 || true
    log_success "Kernel otimizado."
}

secure_bootstrap() {
    log_step "Gerando Identidade e Vault"
    if [ -f ".env" ]; then
        log_warn "Arquivo .env preservado."
        return
    fi

    local DB_PASS=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    local JWT_SEC=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64)
    local VLT_TOK=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 48)

    cat <<EOF > .env
PROJECT_NAME=cascata
NODE_ENV=production
DB_USER=cascata_admin
DB_PASS=${DB_PASS}
DB_NAME=cascata_meta
DRAGONFLY_PORT=6379
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=${VLT_TOK}
SYSTEM_JWT_SECRET=${JWT_SEC}
EOF
    chmod 600 .env
    log_success "Ambiente .env gerado."
}

# --- 6. LAUNCH ---
launch_cluster() {
    log_step "Orquestrando Cluster (Docker Compose)"
    
    if [ ! -f "docker-compose.yml" ]; then
        log_error "docker-compose.yml não encontrado em $(pwd)"
    fi

    log_info "Limpando conflitos..."
    "${DOCKER_CMD[@]}" down --remove-orphans >/dev/null 2>&1 || true

    log_info "Iniciando orquestração v1.0.0.0..."
    "${DOCKER_CMD[@]}" pull -q || true
    "${DOCKER_CMD[@]}" up -d --build

    log_info "Bootstrapping Healthchecks..."
    local timer=0
    while : ; do
        local healthy=$(docker ps --filter "name=cascata" --filter "health=healthy" --format "{{.Names}}" | wc -l)
        local total=$(docker ps --filter "name=cascata" --format "{{.Names}}" | wc -l)
        
        if [ "$healthy" -ge "$total" ] && [ "$total" -gt 0 ]; then
            break
        fi
        
        echo -ne "  ${C_DIM}Sincronizando malha: ${healthy}/${total}...${C_RESET}\r"
        sleep 2
        ((timer++))
        if [ $timer -gt 40 ]; then
            log_warn "Timeout parcial detectado."
            break
        fi
    done
}

# --- 7. COMPLETE ---
show_final() {
    local EXTERNAL_IP=$(curl -s -m 5 ifconfig.me || echo "localhost")
    log_step "CASCATA INICIALIZADO COM SUCESSO"
    echo -e "  ✦ ${C_BOLD}Dashboard Portal:${C_RESET} http://${EXTERNAL_IP}:3000"
    echo -e "  ✦ ${C_BOLD}API Core Handler:${C_RESET} http://${EXTERNAL_IP}:8080"
    echo -e "  ✦ ${C_BOLD}Security Vault Console:${C_RESET} http://${EXTERNAL_IP}:8200\n"
    log_success "Deploy concluído. Orquestrador online.\n"
}

# --- FLOW ---
print_banner
check_privileges
sync_repository
ensure_dependencies
apply_tuning
secure_bootstrap
launch_cluster
show_final
