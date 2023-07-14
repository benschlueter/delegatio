variable "name" {
  type        = string
  default     = "delegatioooo"
  description = "Base name of the cluster."
}

variable "debug" {
  type        = bool
  default     = true
  description = "Enable debug mode. This opens up a debugd port that can be used to deploy a custom bootstrapper."
}

variable "control_plane_count" {
  type        = number
  default     = 1
  description = "The number of control plane nodes to deploy."
}

variable "worker_count" {
  type        = number
  default     = 1
  description = "The number of worker nodes to deploy."
}

variable "state_disk_size" {
  type        = number
  default     = 30
  description = "The size of the state disk in GB."
}

variable "instance_type" {
  type        = string
  default     = "e2-standard-2"
  description = "The GCP instance type to deploy."
}

variable "state_disk_type" {
  type        = string
  default     = "pd-ssd"
  description = "The type of the state disk."
}

variable "image_id" {
  type        = string
  default     = "https://www.googleapis.com/compute/v1/projects/delegatio/global/images/gcp-0-0-0-test"
  description = "The GCP image to use for the cluster nodes."
}

variable "project" {
  type        = string
  description = "The GCP project to deploy the cluster in."
  default     = "delegatio"
}

variable "region" {
  type        = string
  description = "The GCP region to deploy the cluster in."
  default = "europe-west6"
}

variable "zone" {
  type        = string
  description = "The GCP zone to deploy the cluster in."
  default   = "europe-west6-a"
}
