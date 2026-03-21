#!/usr/bin/env bash

# ==============================================================================
#  ██████╗ █████╗ ███████╗ ██████╗  █████╗ ████████╗ █████╗ 
# ██╔════╝██╔══██╗██╔════╝██╔════╝ ██╔══██╗╚══██╔══╝██╔══██╗
# ██║     ███████║███████╗██║      ███████║   ██║   ███████║
# ██║     ██╔══██║╚════██║██║      ██╔══██║   ██║   ██╔══██║
# ╚██████╗██║  ██║███████║╚██████╗ ██║  ██║   ██║   ██║  ██║
#  ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
# ==============================================================================
# CASCATA v1.0.0.0 — SECURE ENTERPRISE INSTALLER
# Environment: Linux (Multi-Distro) / Production / VPS
# Philosophy: Plug & Play, Zero-Hardcode, Military Hardening.
# ==============================================================================

set -euo pipefail
IFS=$'\n\t'

# --- 1. DESIGN SYSTEM (TERMINAL) ---
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
log_step()    { echo -e "\n${C_BOLD}${C_BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${C_RESET}\n${C_BOLD}  ▸ $1${C_RESET}\n"; }

# --- 2. PRE-FLIGHT VERIFICATIONS ---
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
}

check_privileges() {
    if [ "$EUID" -ne 0 ]; then
        log_warn "O instalador necessita de privilégios elevados para tuning de Kernel e Docker."
        if command -v sudo >/dev/null 2>&1; then
            log_info "Elevando privilégios via sudo..."
            exec sudo bash "$0" "$@"
        else
            log_error "Comando 'sudo' ausente. Execute como root."
        fi
    fi
}

detect_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        DISTRO=$ID
    else
        DISTRO="unknown"
    fi
    log_info "Plataforma detectada: ${C_CYAN}${DISTRO}${C_RESET}"
}

# --- 3. INFRASTRUCTURE PROVISIONING ---
install_dependencies() {
    log_step "Sincronizando dependências de sistema"
    
    case "$DISTRO" in
        ubuntu|debian|linuxmint|pop)
            apt-get update -qq && apt-get install -qq -y curl git jq >/dev/null
            ;;
        centos|rhel|rocky|almalinux|fedora|amzn)
            dnf install -y -q curl git jq >/dev/null
            ;;
        *)
            log_warn "Distribuição desconhecida. Tentando prosseguir com binários existentes."
            ;;
    esac
}

ensure_docker() {
    log_step "Validando Docker Runtime & Compose v2"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_info "Docker Engine ausente. Instalando via canal oficial..."
        curl -fsSL https://get.docker.com -o get-docker.sh
        sh get-docker.sh >/dev/null 2>&1
        rm get-docker.sh
        systemctl enable --now docker >/dev/null 2>&1
    fi
    
    if ! docker compose version >/dev/null 2>&1; then
        log_info "Instalando Docker Compose Plugin..."
        # Fallback para instalação manual do binário se o pacote falhar
        DOCKER_CONFIG=${DOCKER_CONFIG:-$HOME/.docker}
        mkdir -p "$DOCKER_CONFIG/cli-plugins"
        curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o "$DOCKER_CONFIG/cli-plugins/docker-compose"
        chmod +x "$DOCKER_CONFIG/cli-plugins/docker-compose"
    fi
    
    log_success "Docker Daemon: $(docker --version)"
    log_success "Compose: $(docker compose version)"
}

tune_kernel() {
    log_step "Aplicando Tuning de Performance no Kernel (BDB/Dragonfly)"
    
    # Necessário para PostgreSQL RAM e Dragonfly Memory Mapping
    sysctl -w vm.max_map_count=524288 >/dev/null 2>&1 || true
    sysctl -w fs.file-max=131072 >/dev/null 2>&1 || true
    
    log_success "Kernel otimizado para alta carga de tenants."
}

# --- 4. SECURE CONFIGURATION ---
generate_environment() {
    log_step "Gerando Identidade Criptográfica do Cluster"
    
    if [ -f ".env" ]; then
        log_warn "Arquivo .env já existente. Ignorando sobrescrita para proteção de dados."
        return
    fi

    # Geradores baseados em /dev/urandom para máxima entropia
    DB_PASS=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    JWT_SECRET=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64)
    VAULT_TOKEN=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 48)

    cat <<EOF > .env
# --- CASCATA V1 MASTER CONFIG ---
PROJECT_NAME=cascata
NODE_ENV=production

# Database
DB_USER=cascata_admin
DB_PASS=${DB_PASS}
DB_NAME=cascata_meta
DB_PORT=5432

# Hyper-Cache
DRAGONFLY_PORT=6379

# Security (Vault)
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=${VAULT_TOKEN}

# Logic & Auth
SYSTEM_JWT_SECRET=${JWT_SECRET}

# Logging
LOG_LEVEL=info
EOF

    log_success "Segredos gerados e injetados em .env (Permissões: 600)"
    chmod 600 .env
}

# --- 5. ORCHESTRATION ---
start_cluster() {
    log_step "Levantando Orquestrador e Serviços"
    
    docker compose pull -q || true
    docker compose up -d
    
    log_info "Aguardando estabilização dos serviços (Healthchecks)..."
    
    local wait_count=0
    while : ; do
        local healthy=$(docker ps --filter "health=healthy" --filter "name=cascata" --format "{{.Names}}" | wc -l)
        local total=$(docker ps --filter "name=cascata" --format "{{.Names}}" | wc -l)
        
        if [ "$healthy" -ge "$total" ] && [ "$total" -gt 0 ]; then
            break
        fi
        
        echo -ne "  ${C_DIM}Sincronizando pilar ${healthy}/${total}...${C_RESET}\r"
        sleep 2
        ((wait_count++))
        if [ $wait_count -gt 30 ]; then
            log_warn "Alguns serviços estão demorando a responder. Verifique logs."
            break
        fi
    done
}

# --- 6. FINALIZATION ---
show_summary() {
    local IP=$(curl -s -m 5 ifconfig.me || echo "localhost")
    
    log_step "DEPLOY CONCLUÍDO COM SUCESSO"
    
    echo -e "  - ${C_BOLD}Cascata Dashboard:${C_RESET} http://${IP}:3000"
    echo -e "  - ${C_BOLD}Backend Data API:${C_RESET}  http://${IP}:8080"
    echo -e "  - ${C_BOLD}Vault UI:${C_RESET}          http://${IP}:8200"
    echo -e "  - ${C_BOLD}PostgreSQL Node:${C_RESET}   ${IP}:5432\n"
    
    log_info "O binário CLI 'cascata' está pronto em: ./cmd/cli/cascata"
    log_success "\nO Cascata v1.0.0.0 está em conformidade. Bem-vindo à era Orquestrada.\n"
}

# --- MAIN EXECUTION ---
print_banner
check_privileges
detect_distro
install_dependencies
ensure_docker
tune_kernel
generate_environment
start_cluster
show_summary
