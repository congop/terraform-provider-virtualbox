variable "apt_proxy" {
  type    = string
  default = "http://192.168.56.1:3142"
}
variable "img_path_or_url" {
  type = string
  description = <<-EOT
  file path or url of the image.
  e.g.
    https://cloud-images.ubuntu.com/releases/focal/release/ubuntu-20.04-server-cloudimg-amd64.ova
    /media/cloudimages/focal-server-cloudimg-amd64.ova
  usage:
    # setting direcly at command line
    TF_VAR_img_path_or_url=/media/cloudimages/focal-server-cloudimg-amd64.ova terraform plan
  EOT
}
variable "k8scluster_nodes" {
  type = map(object({
    hostname      = string
    ipv4_address  = string
    network       = string
    type          = string
    memory_mibs   = number
    mac_address   = string
  }))

  default = {
    k8s_controller1 = {
      hostname    = "controller1.sapone.k8s"
      type        = "controller"
      ipv4_address  = "192.168.56.51"
      network     = "vboxnet0"
      memory_mibs = 2048
      mac_address = "16:2e:99:92:26:13"
    }

  }
}