ssh_user             = "opensuse"
ssh_private_key_path = "~/.ssh/st-ed25519"
nodes = [
  [
    {
      fqdn = "10-0-0-100.sslip.io"
      name = "up-0"
    },
    {
      fqdn = "10-0-0-101.sslip.io"
      name = "up-1"
    },
    {
      fqdn = "10-0-0-102.sslip.io"
      name = "up-2"
    }
  ],
  [
    {
      fqdn = "10-0-0-10.sslip.io"
      name = "down-0"
    }
  ],
  [
    {
      fqdn = "10-0-0-200.sslip.io"
      name = "test-0"
    }
  ]
]
