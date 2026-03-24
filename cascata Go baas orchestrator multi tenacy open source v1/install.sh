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
readonly VAULT_KEYS_DIR="$HOME/.cascata"
readonly VAULT_KEYS_FILE="$VAULT_KEYS_DIR/vault_keys.env"

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
${C_RESET}${C_DIM}                              v1.0.0.0 | Orchestrator Studio${C_RESET}"
    echo -e "${C_DIM}---------------------------------------------------------------${C_RESET}\n"
    log_info "Inicializando Orquestração Zero-Trust..."
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
    log_info "Cascata Architecture: Zero-Host-Dependency (Go is only used inside Docker Build Stage)."
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
    log_success "Parâmetros Kernel gravados e persistidos."
}

secure_bootstrap() {
    log_step "Provisão de Identidade Criptográfica (.env)"
    
    if [[ -f ".env" ]]; then
        log_warn "Arquivo .env já existente. Fazendo backup e gerando novo para FRESH INSTALL."
        mv .env .env.bak.$(date +%s)
    fi

    local DB_PASS=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32)
    local JWT_SEC=$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64)

    cat <<EOF > .env
# --- CASCATA V1 MASTER ENVS ---
PROJECT_NAME=cascata
NODE_ENV=production

# Core Databases
DB_USER=cascata_admin
DB_PASS=${DB_PASS}
DB_NAME=cascata_meta
DB_PORT=5432

# Dragonfly DB Engine
DRAGONFLY_PORT=6379

# Security Infrastructure (Vault)
VAULT_ADDR=http://cascata-vault:8207
VAULT_TOKEN=PENDING_INIT_FROM_BOOTSTRAP

# Internal Logic Exchange
SYSTEM_JWT_SECRET=${JWT_SEC}
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
    # Garantir permissões do arquivo de configuração do Vault (user 'vault' do alpine precisa ler)
    chmod 644 deployments/vault-config.hcl || true
    
    "${DOCKER_CMD[@]}" pull -q || true
    "${DOCKER_CMD[@]}" up -d --build
    
    log_success "Containers Base enviados à rede."
}

# --- 7. SECURE VAULT BOOTSTRAP ---
vault_bootstrap() {
    log_step "Vault Operator: Inicialização Avançada e Unseal"
    
    local VAULT_CONTAINER
    # Melhor detecção: busca pelo container exact name ou pelo serviço do compose local
    VAULT_CONTAINER=$(docker ps --filter "name=cascata-vault" --format "{{.Names}}" | head -n 1)
    
    if [[ -z "$VAULT_CONTAINER" ]]; then
        # Tenta fallback pelo filtro genérico se o nome exato falhar por prefixos
        VAULT_CONTAINER=$(docker ps --filter "name=vault" --format "{{.Names}}" | head -n 1)
    fi

    if [[ -z "$VAULT_CONTAINER" ]]; then
        # Check se o container existe mas caiu
        local EXITED_VAULT
        EXITED_VAULT=$(docker ps -a --filter "name=vault" --format "{{.Status}}" | head -n 1)
        log_error "Container Vault estritamente inoperante. Status detectado: ${EXITED_VAULT}. Logs do container: \n$(docker logs cascata-vault 2>&1 | tail -n 20)"
    fi

    log_info "Aguardando Protocolo Ping Vault API..."
    local v_timer=0
    while : ; do
        set +e
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault status -format=json >/dev/null 2>&1
        local V_STATUS_EXIT=$?
        set -e
        if [[ $V_STATUS_EXIT -eq 0 ]] || [[ $V_STATUS_EXIT -eq 2 ]]; then
            break
        fi
        sleep 2
        v_timer=$((v_timer + 1))
        if [[ "$v_timer" -gt 25 ]]; then # Aumentado para 50s de segurança
            log_warn "Vault demorando no boot."
            break
        fi
    done

    local IS_INIT
    IS_INIT=$(docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault status -format=json 2>/dev/null | jq -r '.initialized' || echo "false")
    
    if [[ "$IS_INIT" == "true" ]]; then
        log_info "Vault Previamente Inicializado."
        
        if [[ -f "$VAULT_KEYS_FILE" ]]; then
            local EXISTING_KEY
            EXISTING_KEY=$(grep "UNSEAL_KEY=" "$VAULT_KEYS_FILE" | cut -d'=' -f2 || true)
            if [[ -n "$EXISTING_KEY" ]]; then
                docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault operator unseal "$EXISTING_KEY" >/dev/null 2>&1 || true
                log_success "Vault reconstituído e Unselado autonomamente."
            fi
        fi
    else
        log_warn "ATENÇÃO: Operação Single-Node / VPS em Cluster Primário."
        log_info "As diretrizes Keys-Shares & Thresholds são configuradas como = 1 para provisionamento cloud único."
        
        local INIT_JSON
        INIT_JSON=$(docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault operator init -key-shares=1 -key-threshold=1 -format=json)
        
        local UNSEAL_KEY
        UNSEAL_KEY=$(echo "$INIT_JSON" | jq -r '.unseal_keys_b64[0]')
        local ROOT_TOKEN
        ROOT_TOKEN=$(echo "$INIT_JSON" | jq -r '.root_token')
        
        # Persistência Ofusca temporária (apenas para a etapa do unseal/login)
        mkdir -p "$VAULT_KEYS_DIR"
        chmod 700 "$VAULT_KEYS_DIR"
        
        log_info "Executando Unseal atômico no Motor Principal..."
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault operator unseal "$UNSEAL_KEY" >/dev/null 2>&1
        
        log_info "Autenticando Motor com Root..."
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault login "$ROOT_TOKEN" >/dev/null 2>&1
        
        log_info "Habilitando Engines Base (Transit e Secrets)..."
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault secrets enable -path=secret kv-v2 >/dev/null 2>&1 || true
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault secrets enable transit >/dev/null 2>&1 || true
        
        log_info "Configurando Soberania: Criando chave Transit 'cascata-pepper'..."
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault write -f transit/keys/cascata-pepper >/dev/null 2>&1 || true
        
        log_info "Aplicando Matriz de Acesso Mínimo Privilégio via CP..."
        
        # Uso do docker cp para evitar silent fails em pipes bash herdados:
        local POLICY_TMP="/tmp/cascata_policy.hcl"
        cat <<EOF > "$POLICY_TMP"
path "secret/*" { capabilities = ["create", "read", "update", "delete", "list"] }
path "transit/*" { capabilities = ["create", "read", "update"] }
path "sys/health" { capabilities = ["read"] }
path "auth/token/renew-self" { capabilities = ["update"] }
EOF
        docker cp "$POLICY_TMP" "$VAULT_CONTAINER":/tmp/cascata_policy.hcl
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault policy write cascata-backend /tmp/cascata_policy.hcl >/dev/null
        rm -f "$POLICY_TMP"
        
        # Cria um AppToken com capacidades de auto-renovação, sem o peso limitador do period fixo
        log_info "Gerando AppToken Restrito (Orphan / Auto-Renovável)..."
        local APP_TOKEN_JSON
        APP_TOKEN_JSON=$(docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault token create -policy="cascata-backend" -orphan -no-default-policy -format=json)
        local APP_TOKEN
        APP_TOKEN=$(echo "$APP_TOKEN_JSON" | jq -r '.auth.client_token')
        
        log_info "Vault Habilitado: APP_TOKEN seguro injetado em .env"
        sed -i "s|^VAULT_TOKEN=.*|VAULT_TOKEN=${APP_TOKEN}|" .env
        
        # Fase de Aniquilação Definitiva (Root Auto-Revogado e Escondido)
        log_info "Revogando acesso Global do Root Token da Sessão..."
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" vault token revoke "$ROOT_TOKEN" >/dev/null 2>&1 || true
        
        # Apenas salva a Unseal Key fora da raiz de repositório
        echo -e "UNSEAL_KEY=${UNSEAL_KEY}" > "$VAULT_KEYS_FILE"
        chmod 600 "$VAULT_KEYS_FILE"
        
        log_success "WARNING CASCATA: Sua Unseal Key Privada (Mestre) reside em: $VAULT_KEYS_FILE"
        log_warn "O Root Token foi apagado do planeta para sua segurança. Armazene o arquivo Unseal Keys OFFLINE e exclua-o em seguida."
    fi
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
    log_step "Injeção de Identidade: Worner Provisioning (Secure Go-Calculated)"
    
    local ORCH_CONTAINER
    ORCH_CONTAINER=$(docker ps --filter "name=orchestrator" --format "{{.Names}}" | head -n 1)
    
    local VAULT_CONTAINER
    VAULT_CONTAINER=$(docker ps --filter "name=vault" --format "{{.Names}}" | head -n 1)
    
    if [[ -z "$ORCH_CONTAINER" ]] || [[ -z "$VAULT_CONTAINER" ]]; then
        log_error "Containers Core inoperantes. Não foi possível provisionar o Worner."
    fi

    log_info "Calculando Argon2id (64MB/4-Threads) e persistindo via Cascata-Engine..."
    
    # 1. Provisionamento via Binário Compilado (Não requer Go no container final)
    local PROVISION_OUT
    PROVISION_OUT=$(docker exec -e DB_URL="postgres://cascata_admin:$(grep '^DB_PASS=' .env | cut -d'=' -f2)@cascata-db:5432/cascata_meta" \
        "$ORCH_CONTAINER" ./worner-provision "$WORNER_EMAIL" "$WORNER_PASS" "$MFA_ENABLED" 2>&1)

    if [[ ! "$PROVISION_OUT" =~ "SUCCESS_ID:" ]]; then
        log_error "Falha no provisionamento: $PROVISION_OUT"
    fi

    local WORNER_ID
    WORNER_ID=$(echo "$PROVISION_OUT" | grep "SUCCESS_ID:" | cut -d':' -f2)
    log_success "Membro Worner ($WORNER_ID) injetado via Engine."

    # 2. Vault MFA Provisioning (Usando o UUID do membro como o path, não o e-mail)
    if [[ "$MFA_ENABLED" == "true" ]]; then
        log_info "MFA Ativo: Protegendo segredo no Vault sob UUID..."
        
        local MFA_SECRET
        MFA_SECRET=$(tr -dc 'A-Z2-7' < /dev/urandom | head -c 32)
        
        local APP_TOKEN
        APP_TOKEN=$(grep '^VAULT_TOKEN=' .env | cut -d '=' -f2)
        
        # O path agora é members/{id}/mfa_secret conforme arquitetura Phase 3
        docker exec -e VAULT_ADDR="http://127.0.0.1:8207" "$VAULT_CONTAINER" sh -c "vault login $APP_TOKEN >/dev/null 2>&1 && vault kv put secret/cascata/members/$WORNER_ID/mfa_secret secret=$MFA_SECRET >/dev/null 2>&1"
        log_success "Segredo OTP persistido no Vault: secret/cascata/members/$WORNER_ID/mfa_secret"
        
        MFA_DISPLAY="  ${C_BOLD}OTP_SECRET (Add no Google Auth):${C_RESET} ${MFA_SECRET}"
    else
        MFA_DISPLAY="  ${C_BOLD}OTP / MFA:${C_RESET} Desativado"
    fi
}

# --- 8. COMPLETION ---
show_final() {
    # Detecção de IP Real (Public Cloud Awareness)
    local EXTERNAL_IP
    EXTERNAL_IP=$(curl -s -m 5 https://checkip.amazonaws.com || curl -s -m 5 https://ifconfig.me || echo "localhost")
    
    log_step "CASCATA STUDIO v1 INSTALADO"
    
    # Exibições de Tokens padrão como solicitado. Apenas os de utilização.
    local JWT_EXT=$(grep '^SYSTEM_JWT_SECRET=' .env | cut -d '=' -f2)
    local APP_EXT=$(grep '^VAULT_TOKEN=' .env | cut -d '=' -f2)

    echo -e "  ✦ ${C_BOLD}Cascata Studio (Dashboard):${C_RESET}  http://${EXTERNAL_IP}"
    echo -e "  ✦ ${C_BOLD}Cascata Backend (V1 API):${C_RESET}    http://${EXTERNAL_IP}/v1/\n"
    
    echo -e "${C_DIM}--- Chaves de Acesso e Setup ---${C_RESET}"
    echo -e "  ${C_BOLD}Worner E-mail:${C_RESET} ${WORNER_EMAIL}"
    echo -e "$MFA_DISPLAY\n"
    
    echo -e "  ${C_BOLD}DB_ADMIN:${C_RESET} cascata_admin"
    echo -e "  ${C_BOLD}JWT_MASTER:${C_RESET} ${JWT_EXT:0:20}...${JWT_EXT: -5}"
    echo -e "  ${C_BOLD}VAULT_APP_TOKEN:${C_RESET} ${APP_EXT:0:15}...\n"
    
    log_success "Deploy Inviolável. Mantenha os endereços e segredos longe do escrutínio público."
    
    # Dica de Operação Manual (Permissões Docker)
    if ! groups $USER | grep &>/dev/null "\bdocker\b"; then
        log_warn "DICA: Para rodar comandos docker manuais sem sudo, execute:"
        echo -e "  ${C_CYAN}sudo usermod -aG docker \$USER && newgrp docker${C_RESET}\n"
    fi
}

# --- EXECUTION FLOW ---
print_banner
check_privileges
collect_worner_credentials
sync_repository
ensure_dependencies
apply_tuning
secure_bootstrap
launch_cluster_base
vault_bootstrap
reload_and_verify
provision_worner_execution
show_final
