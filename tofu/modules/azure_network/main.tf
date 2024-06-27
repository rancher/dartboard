resource "azurerm_virtual_network" "main" {
  name                = "${var.project_name}-network"
  address_space       = ["172.16.0.0/12"]
  location            = var.location
  resource_group_name = var.resource_group_name
}

resource "azurerm_subnet" "public" {
  depends_on           = [azurerm_virtual_network.main]
  name                 = "${var.project_name}-public-subnet"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["172.16.0.0/24"]
}

resource "azurerm_subnet" "private" {
  depends_on           = [azurerm_virtual_network.main]
  name                 = "${var.project_name}-private-subnet"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["172.20.0.0/16"]
}

resource "azurerm_public_ip" "public" {
  name                = "${var.project_name}-public-ip"
  location            = var.location
  resource_group_name = var.resource_group_name
  allocation_method   = "Static"
}

resource "azurerm_network_security_group" "public" {
  name                = "${var.project_name}-sg"
  location            = var.location
  resource_group_name = var.resource_group_name

  security_rule {
    name                       = "${var.project_name}-ingress-01"
    description                = "Allow inbound connections on port 22"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
  security_rule {
    name                       = "${var.project_name}-ingress-02"
    description                = "Allow inbound connections from the private subnet"
    priority                   = 101
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefixes    = azurerm_subnet.private.address_prefixes
    destination_address_prefix = "*"
  }
  security_rule {
    name                       = "${var.project_name}-egress-01"
    description                = "Allow outbound connections"
    priority                   = 100
    direction                  = "Outbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "public" {
  subnet_id                 = azurerm_subnet.public.id
  network_security_group_id = azurerm_network_security_group.public.id
}

resource "azurerm_ssh_public_key" "key_pair" {
  name                = "${var.project_name}-key-pair"
  resource_group_name = var.resource_group_name
  location            = var.location
  public_key          = file(var.ssh_public_key_path)
}

module "bastion" {
  source              = "../azure_host"
  project_name        = var.project_name
  name                = "bastion"
  resource_group_name = var.resource_group_name
  location            = var.location
  os_image = {
    publisher = "suse"
    offer     = "opensuse-leap-15-5"
    sku       = "gen2"
    version   = "latest"
  }
  subnet_id            = azurerm_subnet.public.id
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
  public_ip_address_id = azurerm_public_ip.public.id
}
