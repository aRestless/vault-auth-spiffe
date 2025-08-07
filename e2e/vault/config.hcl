disable_mlock = true
ui = true

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = 1
}

storage "inmem" {}

# The following is required for the SPIFFE auth method to be available
plugin_directory = "/vault/plugins"

log_level = "Debug"