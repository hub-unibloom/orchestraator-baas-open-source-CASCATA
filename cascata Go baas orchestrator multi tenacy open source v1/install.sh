#!/usr/bin/env bash

# ==============================================================================
#  ██████╗ █████╗ ███████╗ ██████╗  █████╗ ████████╗ █████╗ 
# ██╔════╝██╔══██╗██╔════╝██╔════╝ ██╔══██╗╚══██╔══╝██╔══██╗
# ██║     ███████║███████╗██║      ███████║   ██║   ███████║
# ██║     ██╔══██║╚════██║██║      ██╔══██║   ██║   ██╔══██║
# ╚██████╗██║  ██║███████║╚██████╗ ██║  ██║   ██║   ██║  ██║
#  ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝
# ==============================================================================
# CASCATA v1.0.0.0 — SECURE ENTERPRISE INSTALLER (HARDENED)
# Environment: Linux (Multi-Distro) / Production / VPS
# Philosophy: Plug & Play, Context-Aware, Military Hardening.
# ==============================================================================

set -euo pipefail
IFS=$'\n\t'

# --- 1. DESIGN SYSTEM & IDENTITY ---
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

# --- 2. CONTEXT & PRE-FLIGHT ---
# Ensures the script runs with absolute directory awareness
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR" || log_error "Falha crítica ao acessar diretório raiz."

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

check_integrity() {
    if [ ! -f "docker-compose.yml" ]; then
        log_error "Aquivo 'docker-compose.yml' ausente em $ROOT_DIR. O instalador deve permanecer na raiz do projeto."
    fi
}

check_privileges() {
    if [ "$EUID" -ne 0 ]; then
        log_warn "O instalador necessita de privilégios elevados para tuning de Kernel e Docker."
        if command -v sudo >/dev/null 2>&1; then
            log_info "Tentando elevar automaticamente via sudo..."
            exec sudo bash "$0" "$@"
        else
            log_error "Permissão negada. Execute: sudo ./install.sh"
        fi
    fi
}

# --- 3. INFRASTRUCTURE & TUNING ---
ensure_dependencies() {
    log_step "Validando Docker Runtime & System Binaries"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_info "Provisionando Docker Engine nativo..."
        curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh && rm get-docker.sh
        systemctl enable --now docker >/dev/null 2>&1 || true
    fi
    
    # Check for Compose v2 or v1
    DOCKER_COMPOSE_CMD="docker compose"
    if ! docker compose version >/dev/null 2>&1; then
        if command -v docker-compose >/dev/null 2>&1; then
            DOCKER_COMPOSE_CMD="docker-compose"
            log_warn "Usando legado: docker-compose (v1). Recomendado v2."
        else
            log_error "Docker Compose não detectado. Instale 'docker-compose-plugin'."
        fi
    fi
    
    log_success "Docker Runtime operacional: $(docker --version)"
}

apply_hardening() {
    log_step "Aplicando Hardening e Tuning de Performance (Kernel)"
    
    # BDB & Dragonfly Optimization
    sysctl -w vm.max_map_count=524288 >/dev/null 2>&1 || true
    sysctl -w fs.file-max=131072 >/dev/null 2>&1 || true
    
    # TCP Keepalives (BaaS Performance)
    sysctl -w net.ipv4.tcp_keepalive_time=60 >/dev/null 2>&1 || true
    
    log_success "Tuning de Kernel concluído."
}

# --- 4. SECURE PROVISIONING ---
secure_env() {
    log_step "Gerando Identidade Criptográfica (Vault Bootstrap)"
    
    if [ -f ".env" ]; then
        log_warn "Identidade .env já existente. Ignorando regeração para preservar chaves ativas."
        return
    fi

    # Entropy-driven keys
    PG_PASS=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    JWT_SECRET=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64)
    VAULT_TOKEN=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 48)

    cat <<EOF > .env
# --- CASCATA V1 MASTER IDENT ---
PROJECT_NAME=cascata
NODE_ENV=production

# Database (Logical Isolator)
DB_USER=cascata_admin
DB_PASS=${PG_PASS}
DB_NAME=cascata_meta

# Networking & Proxy
DRAGONFLY_PORT=6379
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=${VAULT_TOKEN}

# Logic & Handshake
SYSTEM_JWT_SECRET=${JWT_SECRET}
EOF
    chmod 600 .env
    log_success "Secrets protegidos em .env (Mode 600)."
}

# --- 5. ORCHESTRATION ---
start_cluster() {
    log_step "Aplicando Configurações do Cluster"
    
    # FAIL-FAST: Kill old zombie containers from this project
    log_info "Matando containeres antigos (Fail Fast se houver conflito)"
    $DOCKER_COMPOSE_CMD down --remove-orphans >/dev/null 2>&1 || true
    
    log_info "Provisionando Persistent Volumes..."
    docker volume create cascata_data >/dev/null 2>&1 || true
    docker volume create cascata_vault >/dev/null 2>&1 || true

    log_info "Orquestrando Pillar Services (-d)..."
    $DOCKER_COMPOSE_CMD up -d --build
    
    log_info "Bootstrapping Healthchecks..."
    local wait_count=0
    while : ; do
        local total_ready=$(docker ps --filter "health=healthy" --filter "name=cascata" --format "{{.Names}}" | wc -l)
        local total_containers=$(docker ps --filter "name=cascata" --format "{{.Names}}" | wc -l)
        
        if [ "$total_ready" -ge "$total_containers" ] && [ "$total_containers" -gt 0 ]; then
            break
        fi
        
        echo -ne "  ${C_DIM}Aguardando estabilização: ${total_ready}/${total_containers}...${C_RESET}\r"
        sleep 2
        ((wait_count++))
        if [ $wait_count -gt 25 ]; then
            log_warn "Timeout parcial. Alguns serviços podem exigir monitoramento manual."
            break
        fi
    done
}

# --- 6. COMPLETION ---
show_final() {
    local EXTERNAL_IP=$(curl -s -m 5 ifconfig.me || echo "127.0.0.1")
    
    log_step "CASCATA INSTALADO COM SUCESSO"
    
    echo -e "  ✦ ${C_BOLD}Front-End Portal:${C_RESET} http://${EXTERNAL_IP}:3000"
    echo -e "  ✦ ${C_BOLD}API Core Handler:${C_RESET} http://${EXTERNAL_IP}:8080"
    echo -e "  ✦ ${C_BOLD}Vault Management:${C_RESET} http://${EXTERNAL_IP}:8200\n"
    
    log_success "Deploy concluído perfeitamente. O orquestrador v1.0.0.0 está online.\n"
}

# --- EXECUTION FLOW ---
print_banner
check_integrity
check_privileges
ensure_dependencies
apply_hardening
secure_env
start_cluster
show_final
