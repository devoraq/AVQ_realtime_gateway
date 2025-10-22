variable "kafka_bootstrap_servers" {
  type        = list(string)
  description = "List of Kafka bootstrap servers; defaults align with docker-compose dev environment."
  default     = ["localhost:9092"]
}

variable "topic_name" {
  type        = string
  description = "Kafka topic name managed for the dev environment."
  default     = "test-topic"
}

variable "topic_partitions" {
  type        = number
  description = "Number of partitions for the dev topic (aligns with KAFKA_NUM_PARTITIONS)."
  default     = 3
}

variable "replication_factor" {
  type        = number
  description = "Replication factor for the dev topic (single broker by default)."
  default     = 1
}

variable "topic_cleanup_policy" {
  type        = string
  description = "Cleanup policy for the dev topic (delete or compact)."
  default     = "delete"
}

variable "topic_min_insync_replicas" {
  type        = number
  description = "Minimum in-sync replicas required for producer acks."
  default     = 1
}

variable "topic_segment_bytes" {
  type        = number
  description = "Segment file size threshold in bytes."
  default     = 104857600
}

variable "extra_topic_config" {
  type        = map(string)
  description = "Additional topic-level configuration overrides."
  default     = {}
}

variable "kafka_sasl_username" {
  type        = string
  description = "Optional SASL username for secured clusters."
  default     = ""
  nullable    = false
}

variable "kafka_sasl_password" {
  type        = string
  description = "Optional SASL password for secured clusters."
  default     = ""
  nullable    = false
  sensitive   = true
}

variable "kafka_sasl_mechanism" {
  type        = string
  description = "Optional SASL mechanism (e.g., PLAIN)."
  default     = ""
  nullable    = false
}
