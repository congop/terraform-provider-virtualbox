
data "template_cloudinit_config" "config" {

  for_each  = var.k8scluster_nodes

  gzip          = false
  base64_encode = false

  # password is: ansible
  part {
    content_type = "text/cloud-config"
    content      = <<-EOT
    users:
      - default
      - name: ansible
        gecos: DevOps Tool
        primary_group: foobar
        groups: users
        lock_passwd: false
        sudo: ALL=(ALL) NOPASSWD:ALL
        passwd: $6$rounds=4096$zucT9vIn$dYNtJdZ5Z4pYfCF9CGLtmX/l5xLGef0WqzihpMTcyfunK3u5qWltdRoWlsy0hbLiYVb3dh46AM.Kbo/4mv52o0
        shell: /bin/bash
    apt:
      proxy: ${var.apt_proxy}
    #package_update: true
    #packages:
    #  - virtualbox-guest-dkms
    #  - virtualbox-guest-x11
    #  - virtualbox-guest-utils
    ssh_pwauth: True
    manage_etc_hosts: true
    fqdn: ${each.value.hostname}
    hostname: ${each.value.hostname}
    runcmd:
      - touch /var/log/cloud-init-virtualbox.log
      - loadkeys de
      - localectl set-keymap de
      - cloud-init-per once do-apt-update                 apt update -y
      - cloud-init-per once do-apt-fullupgrade            apt full-upgrade -y
      - cloud-init-per once do-apt-install-inotify-tools  apt install -y inotify-tools --no-install-recommends
      - cloud-init-per once do-apt-install-guestadd       apt install -y virtualbox-guest-dkms virtualbox-guest-x11 virtualbox-guest-utils --no-install-recommends
      - cloud-init-per once do-apt-autoremove             apt autoremove --purge -y
      - modprobe vboxsf
      - modprobe vboxguest
      - "systemctl restart virtualbox-guest-utils.service | true"
      - exit 0
      #- nohup /var/lib/cloud/instance/scripts/shareCloudInitFinalStatusAsVirtualBoxGuestProperties.sh &>/var/log/cloud-init-virtualbox.log &
      #- VBoxControl guestproperty set "/VirtualBox/GuestInfo/CloudInit/Status" "$(cloud-init status)"
    EOT
  }

  #Main cloud-config configuration file.
  part {
    filename     = "shareCloudInitFinalStatusAsVirtualBoxGuestProperties.sh"
    content_type = "text/x-shellscript"
    content      = templatefile("shareCloudInitFinalStatusAsVirtualBoxGuestProperties.sh", {})
  }
}

resource "virtualbox_vm" "node" {
  # memory    = "1024 mib"

  depends_on = [data.template_cloudinit_config.config]

  for_each  = var.k8scluster_nodes
  name      = each.key
  memory    = "${each.value.memory_mibs} mib"

  image     = var.img_path_or_url
  cpus      = 2
  user_data = jsonencode({
    role = "${each.value.type}"
  })

  creation_policy {
    type    = "cloud_init_done_by_vm_guestproperty"
		timeout = "PT5M"
  }

  network_adapter {
    type           = "hostonly"
    host_interface = each.value.network
    #host_interface = "vboxnet1"
    mac_address    = replace(each.value.mac_address, ":","")
  }

  cloud_init {
    user_data = data.template_cloudinit_config.config[each.key].rendered

    meta_data = <<-EOT
    instance-id: ${each.key}
    hostname: ${each.value.hostname}
    local_hostname: ${each.value.hostname}
    EOT

    network_config = <<-EOT
    version: 2
    ethernets:
      interface0:
        match:
          macaddress: ${each.value.mac_address}
        set-name: eth0
        addresses:
          - ${each.value.ipv4_address}/24
        gateway4: 192.168.56.1
    EOT

  }

  #uart, err := NewUART("uart1", "16550A", "0x2f8", "3", "file", "/tmp/uart1")
  uart {
    key       = "uart1"
    port      = "0x03f8"
    irq       = "4"
    type      = "16550A"
    mode      = "file"
    mode_data = "/tmp/uart1"
  }
}

# output "IPAddr" {
#   #value = element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 1)
#   value = virtualbox_vm.node.*.network_adapter.0.ipv4_address
# }
# # output "IPAddr_2" {
# #   value = element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 2)
# # }

# output Nics {
#   value =  [for adapter in flatten(virtualbox_vm.node[*].network_adapter):
#     {status=adapter.status, ip=adapter.ipv4_address}
#   ]
# }


output Nics {
  value =  [for adapter in flatten(virtualbox_vm.node[*]):
    adapter
  ]
}

