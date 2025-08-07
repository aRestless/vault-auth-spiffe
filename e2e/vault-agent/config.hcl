pid_file = "./pidfile"

auto_auth {
    method "spiffe" {
        mount_path = "auth/jwt"
        config = {
            role = "my-role"
            audience = "vault"
        }
    }
}

cache {
    use_auto_auth_token = true
}

listener "tcp" {
    address = "127.0.0.1:8100"
    tls_disable = true
}
log_level = "Debug"

template {
    source = "/vault/config/test-secret.tmpl"
    destination = "/output/test-secret"
}