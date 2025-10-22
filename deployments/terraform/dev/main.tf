terraform {
  required_version = ">= 1.5.0"

  required_providers {
    kafka = {
      source  = "Mongey/kafka"
      version = "~> 0.5"
    }
  }
}

provider "kafka" {
  bootstrap_servers = var.kafka_bootstrap_servers
  tls_enabled       = false
  skip_tls_verify   = true
  sasl_username     = trimspace(var.kafka_sasl_username) != "" ? var.kafka_sasl_username : null
  sasl_password     = trimspace(var.kafka_sasl_password) != "" ? var.kafka_sasl_password : null
  sasl_mechanism    = trimspace(var.kafka_sasl_mechanism) != "" ? var.kafka_sasl_mechanism : null
}

locals {
  topic_config = merge(
    {
      "cleanup.policy"      = var.topic_cleanup_policy
      "min.insync.replicas" = tostring(var.topic_min_insync_replicas)
      "segment.bytes"       = tostring(var.topic_segment_bytes)
    },
    var.extra_topic_config
  )
}

resource "kafka_topic" "dev" {
  name               = var.topic_name
  replication_factor = var.replication_factor
  partitions         = var.topic_partitions

  config = { for k, v in local.topic_config : k => v if v != "" }
}
