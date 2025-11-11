variable "kubeconfig" {
  description = "Path to kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace for Cortex"
  type        = string
  default     = "cortex"
}

variable "host" {
  description = "Ingress hostname for Cortex frontend"
  type        = string
  default     = "cortex.example.com"
}

variable "frontend_image" {
  description = "Frontend container image"
  type        = string
  default     = "ghcr.io/your-org/cortex-frontend:latest"
}

variable "backend_image" {
  description = "Backend container image"
  type        = string
  default     = "ghcr.io/your-org/cortex-backend:latest"
}

variable "redis_addr" {
  description = "Redis address (host:port)"
  type        = string
  default     = "redis.default.svc.cluster.local:6379"
}

variable "api_key" {
  description = "Cortex API Key"
  type        = string
  sensitive   = true
}

