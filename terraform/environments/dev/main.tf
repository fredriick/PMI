terraform {
  required_version = ">= 1.0"
}

module "k8s_cluster" {
  source = "../../modules/k8s-cluster"

  region       = "us-east-1"
  subnet_ids   = ["subnet-xxx", "subnet-yyy"]
  node_count   = 2
  cluster_name = "proxymesh-dev"
}

module "redis" {
  source = "../../modules/redis"

  region                   = "us-east-1"
  subnet_ids               = ["subnet-xxx", "subnet-yyy"]
  redis_security_group_id  = "sg-xxx"
  redis_auth_token         = var.redis_password
  cluster_name             = "proxymesh-dev"
}

variable "redis_password" {
  type      = string
  sensitive = true
}

output "k8s_cluster_endpoint" {
  value = module.k8s_cluster.cluster_endpoint
}

output "redis_endpoint" {
  value = module.redis.redis_endpoint
}
