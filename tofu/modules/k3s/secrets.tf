resource "tls_private_key" "client_ca_key" {
  algorithm = "ED25519"
}

resource "tls_self_signed_cert" "client_ca_cert" {
  validity_period_hours = 876600 # 100 years
  allowed_uses          = ["digital_signature", "key_encipherment", "cert_signing"]
  private_key_pem       = tls_private_key.client_ca_key.private_key_pem
  is_ca_certificate     = true

  subject {
    common_name = "client-ca-cert"
  }
}

resource "tls_private_key" "server_ca_key" {
  algorithm = "ED25519"
}

resource "tls_self_signed_cert" "server_ca_cert" {
  validity_period_hours = 876600 # 100 years
  allowed_uses          = ["digital_signature", "key_encipherment", "cert_signing"]
  private_key_pem       = tls_private_key.server_ca_key.private_key_pem
  is_ca_certificate     = true

  subject {
    common_name = "server-ca-cert"
  }
}

resource "tls_private_key" "request_header_ca_key" {
  algorithm = "ED25519"
}

resource "tls_self_signed_cert" "request_header_ca_cert" {
  validity_period_hours = 876600 # 100 years
  allowed_uses          = ["digital_signature", "key_encipherment", "cert_signing"]
  private_key_pem       = tls_private_key.request_header_ca_key.private_key_pem
  is_ca_certificate     = true

  subject {
    common_name = "request-header-ca-cert"
  }
}

resource "tls_private_key" "master_user" {
  algorithm = "ED25519"
}

resource "tls_cert_request" "master_user" {
  private_key_pem = tls_private_key.master_user.private_key_pem

  subject {
    common_name  = "master-user"
    organization = "system:masters"
  }
}

resource "tls_locally_signed_cert" "master_user" {
  cert_request_pem   = tls_cert_request.master_user.cert_request_pem
  ca_private_key_pem = tls_private_key.client_ca_key.private_key_pem
  ca_cert_pem        = tls_self_signed_cert.client_ca_cert.cert_pem

  validity_period_hours = 876600

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "client_auth"
  ]
}
