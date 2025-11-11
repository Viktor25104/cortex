resource "kubernetes_namespace" "cortex" {
  metadata {
    name = var.namespace
    labels = { name = var.namespace }
  }
}

resource "kubernetes_config_map" "config" {
  metadata {
    name      = "cortex-config"
    namespace = kubernetes_namespace.cortex.metadata[0].name
  }
  data = {
    REDIS_ADDR = var.redis_addr
  }
}

resource "kubernetes_secret" "backend" {
  metadata {
    name      = "cortex-backend-secrets"
    namespace = kubernetes_namespace.cortex.metadata[0].name
  }
  data = {
    CORTEX_API_KEY = var.api_key
  }
  type = "Opaque"
}

resource "kubernetes_deployment" "frontend" {
  metadata {
    name      = "cortex-frontend"
    namespace = kubernetes_namespace.cortex.metadata[0].name
    labels = { app = "cortex-frontend" }
  }
  spec {
    replicas = 2
    selector { match_labels = { app = "cortex-frontend" } }
    template {
      metadata { labels = { app = "cortex-frontend" } }
      spec {
        container {
          name  = "nginx"
          image = var.frontend_image
          port { container_port = 80 }
          resources {
            requests = { cpu = "50m", memory = "64Mi" }
            limits   = { cpu = "250m", memory = "256Mi" }
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "frontend" {
  metadata {
    name      = "cortex-frontend"
    namespace = kubernetes_namespace.cortex.metadata[0].name
    labels    = { app = "cortex-frontend" }
  }
  spec {
    selector = { app = "cortex-frontend" }
    port { name = "http" port = 80 target_port = 80 }
    type = "ClusterIP"
  }
}

resource "kubernetes_deployment" "backend" {
  metadata {
    name      = "cortex-backend"
    namespace = kubernetes_namespace.cortex.metadata[0].name
    labels    = { app = "cortex-backend" }
  }
  spec {
    replicas = 2
    selector { match_labels = { app = "cortex-backend" } }
    template {
      metadata { labels = { app = "cortex-backend" } }
      spec {
        container {
          name  = "api"
          image = var.backend_image
          port { container_port = 8080 }
          env {
            name = "CORTEX_API_KEY"
            value_from {
              secret_key_ref { name = kubernetes_secret.backend.metadata[0].name key = "CORTEX_API_KEY" }
            }
          }
          env {
            name = "REDIS_ADDR"
            value_from {
              config_map_key_ref { name = kubernetes_config_map.config.metadata[0].name key = "REDIS_ADDR" }
            }
          }
          resources {
            requests = { cpu = "100m", memory = "128Mi" }
            limits   = { cpu = "500m", memory = "512Mi" }
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "backend" {
  metadata {
    name      = "cortex-backend"
    namespace = kubernetes_namespace.cortex.metadata[0].name
    labels    = { app = "cortex-backend" }
  }
  spec {
    selector = { app = "cortex-backend" }
    port { name = "http" port = 8080 target_port = 8080 }
    type = "ClusterIP"
  }
}

resource "kubernetes_ingress_v1" "ingress" {
  metadata {
    name      = "cortex-ingress"
    namespace = kubernetes_namespace.cortex.metadata[0].name
    annotations = {
      "kubernetes.io/ingress.class" = "nginx"
      "cert-manager.io/cluster-issuer" = "letsencrypt"
    }
  }
  spec {
    rule {
      host = var.host
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend { service { name = kubernetes_service.frontend.metadata[0].name port { number = 80 } } }
        }
      }
    }
    tls { secret_name = "cortex-tls" host = var.host }
  }
}

resource "kubernetes_network_policy" "default_deny" {
  metadata { name = "default-deny-all" namespace = kubernetes_namespace.cortex.metadata[0].name }
  spec {
    pod_selector {}
    policy_types = ["Ingress", "Egress"]
  }
}

resource "kubernetes_network_policy" "backend_allow" {
  metadata { name = "backend-allow-from-frontend-and-ingress" namespace = kubernetes_namespace.cortex.metadata[0].name }
  spec {
    pod_selector { match_labels = { app = "cortex-backend" } }
    policy_types = ["Ingress", "Egress"]
    ingress { from { pod_selector { match_labels = { app = "cortex-frontend" } } } }
    ingress { from { namespace_selector { match_labels = { "kubernetes.io/metadata.name" = "ingress-nginx" } } } }
    egress { ports { port = 53 protocol = "UDP" } }
    egress { ports { port = 53 protocol = "TCP" } }
    egress {
      to { namespace_selector {} pod_selector { match_labels = { "app.kubernetes.io/name" = "redis" } } }
      ports { port = 6379 protocol = "TCP" }
    }
  }
}

