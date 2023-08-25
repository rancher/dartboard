ssh_user             = "opensuse"
ssh_private_key_path = "~/.ssh/st-ed25519"
nodes = [
  [
    {
      addr = "10.0.0.100"
      name = "up-0"
    },
    {
      addr = "10.0.0.101"
      name = "up-1"
    },
    {
      addr = "10.0.0.102"
      name = "up-2"
    }
  ],
  [
    {
      addr = "10.0.0.10"
      name = "down-0"
    }
  ],
  [
    {
      addr = "10.0.0.200"
      name = "test-0"
    }
  ]
]
