terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  default = "us-east-1"
}

variable "cluster_name" {
  default = "proxymesh"
}

variable "node_count" {
  default = 2
}

resource "aws_elasticache_subnet_group" "proxymesh" {
  name       = "proxymesh-redis-subnet"
  subnet_ids = var.subnet_ids
}

resource "aws_elasticache_replication_group" "proxymesh" {
  replication_group_id          = "${var.cluster_name}-redis"
  description                   = "ProxyMesh Redis cluster"
  engine                        = "redis"
  engine_version                = "7.1"
  node_type                     = "cache.t3.micro"
  port                          = 6379
  parameter_group_name          = "default.redis7"
  num_cache_clusters            = 2
  automatic_failover_enabled    = true
  multi_az_enabled              = true
  subnet_group_name             = aws_elasticache_subnet_group.proxymesh.name
  security_group_ids            = [var.redis_security_group_id]
  at_rest_encryption_enabled    = true
  transit_encryption_enabled    = true
  auth_token                    = var.redis_auth_token

  tags = {
    Name = "${var.cluster_name}-redis"
  }
}

output "redis_endpoint" {
  value = aws_elasticache_replication_group.proxymesh.primary_endpoint_address
}

output "redis_port" {
  value = aws_elasticache_replication_group.proxymesh.port
}
