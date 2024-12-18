resource "azurerm_resource_group" "rg" {
  name     = "${var.project_name}-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_virtual_network" "main" {
  name                = "${var.project_name}-network"
  address_space       = ["172.16.0.0/12"]
  location            = var.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_subnet" "public" {
  depends_on           = [azurerm_virtual_network.main]
  name                 = "${var.project_name}-public-subnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["172.16.0.0/24"]
}

resource "azurerm_subnet" "private" {
  depends_on           = [azurerm_virtual_network.main]
  name                 = "${var.project_name}-private-subnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["172.20.0.0/16"]
}

resource "azurerm_network_security_group" "public" {
  name                = "${var.project_name}-sg"
  location            = var.location
  resource_group_name = azurerm_resource_group.rg.name

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

resource "azurerm_storage_account" "storage_account" {
  name                     = "${replace(var.project_name, "/[^0-9a-z]/", "")}sa"
  resource_group_name      = azurerm_resource_group.rg.name
  location                 = var.location
  account_replication_type = "LRS"
  account_tier             = "Standard"
  tags = {
    project = var.project_name
  }
}

resource "azurerm_ssh_public_key" "key_pair" {
  name                = "${var.project_name}-key-pair"
  resource_group_name = azurerm_resource_group.rg.name
  location            = var.location
  public_key          = file(var.ssh_public_key_path)
}

module "bastion" {
  source               = "../node"
  project_name         = var.project_name
  name                 = "bastion"
  ssh_private_key_path = var.ssh_private_key_path
  ssh_user             = var.ssh_bastion_user
  public = true
  backend_variables = {
    os_image = var.bastion_os_image
    size     = "Standard_D2s_v4"
    is_spot  = false
    os_disk_type = "StandardSSD_LRS"
    os_disk_size = 30
    os_ephemeral_disk = false
  }

  network_backend_variables = {
    location             = var.location
    resource_group_name  = azurerm_resource_group.rg.name
    public_subnet_id : azurerm_subnet.public.id,
    private_subnet_id : azurerm_subnet.private.id,
    ssh_public_key_path : var.ssh_public_key_path,
    ssh_bastion_host : null,
    ssh_bastion_user : null,
    storage_account_uri : null,
  }
}
