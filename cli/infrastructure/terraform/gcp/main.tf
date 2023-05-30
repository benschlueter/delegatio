terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.65.1"
    }
  }
}

provider "google" {
  credentials = file("delegatio.json")

  project = "delegatio"
  region  = "europe-west6"
  zone    = "europe-west6-a"
}

/* resource "google_compute_image" "example" {
  name = "example-image"

  raw_disk {
    source = "https://storage.cloud.google.com/delegatio-image-23-05-23/image.raw"
  }
} */

resource "google_compute_instance" "default" {
  name         = "test"
  machine_type = "e2-medium"

  tags = ["foo", "bar"]

  boot_disk {
    initialize_params {
      image = "https://storage.cloud.google.com/delegatio-image-23-05-23/image.raw"
      labels = {
        my_label = "value"
      }
    }
  }

  network_interface {
    network = "default"
    access_config {
      // Ephemeral public IP
    }
  }

  metadata = {
    foo = "bar"
  }
}


resource "google_compute_network" "vpc_network" {
  name = "terraform-network"
}