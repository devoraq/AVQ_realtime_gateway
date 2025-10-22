output "topic_name" {
  description = "Kafka topic managed by this Terraform stack."
  value       = kafka_topic.dev.name
}
