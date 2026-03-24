# Cascata Vault Production Config
storage "file" {
  path = "/vault/file"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1
}

ui = true
api_addr = "http://cascata-vault:8200"
disable_mlock = true
