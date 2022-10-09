group "default" {
  targets = ["build"]
}

target "docker-metadata-action" {}

target "build" {
  inherits = ["docker-metadata-action"]
  dockerfile = "Dockerfile"
  platforms = ["linux/amd64", "linux/arm64", "linux/arm/v7"]
}
