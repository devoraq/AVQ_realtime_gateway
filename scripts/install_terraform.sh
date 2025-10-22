#!/bin/sh
set -eu

if command -v terraform >/dev/null 2>&1; then
    echo "Terraform already installed: $(terraform version | head -n 1)"
    INSTALL_NEEDED=false
else
    INSTALL_NEEDED=true
fi

if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo "Run as root or install sudo." >&2
        exit 1
    fi
else
    SUDO=""
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
TERRAFORM_DIR="$PROJECT_ROOT/deployments/terraform/dev"

if [ ! -r /etc/os-release ]; then
    echo "Unable to detect distribution (missing /etc/os-release)." >&2
    exit 1
fi

. /etc/os-release

install_debian_like() {
    echo "Installing Terraform via HashiCorp APT repository..."
    $SUDO apt-get update
    $SUDO apt-get install -y curl gnupg software-properties-common lsb-release
    # Ensure keyring directory exists with proper permissions.
    $SUDO install -d -m 0755 /usr/share/keyrings
    curl -fsSL https://apt.releases.hashicorp.com/gpg | $SUDO gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
    release_codename=$(lsb_release -cs)
    printf "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com %s main\n" "$release_codename" \
        | $SUDO tee /etc/apt/sources.list.d/hashicorp.list >/dev/null
    $SUDO apt-get update
    $SUDO apt-get install -y terraform
}

install_arch_like() {
    echo "Installing Terraform via pacman..."
    $SUDO pacman -Syu --needed --noconfirm terraform
}

case "${ID:-}" in
    ubuntu|debian|linuxmint|pop|elementary|zorin)
        if [ "$INSTALL_NEEDED" = true ]; then
            install_debian_like
        fi
        ;;
    arch|manjaro|endeavouros|arco|archarm)
        if [ "$INSTALL_NEEDED" = true ]; then
            install_arch_like
        fi
        ;;
    *)
        if echo "${ID_LIKE:-}" | grep -Eq 'debian|ubuntu'; then
            if [ "$INSTALL_NEEDED" = true ]; then
                install_debian_like
            fi
        elif echo "${ID_LIKE:-}" | grep -Eq 'arch'; then
            if [ "$INSTALL_NEEDED" = true ]; then
                install_arch_like
            fi
        else
            echo "This script supports only Debian/Ubuntu and Arch-based systems." >&2
            exit 1
        fi
        ;;
esac

if [ "$INSTALL_NEEDED" = true ]; then
    echo "Terraform installed:"
    terraform version | head -n 1
fi

ensure_kafka_running() {
    compose_file="$PROJECT_ROOT/docker-compose.yaml"
    if [ ! -f "$compose_file" ]; then
        echo "docker-compose.yaml not found at $compose_file; skipping Kafka startup." >&2
        return
    fi

    if command -v docker >/dev/null 2>&1; then
        if docker compose version >/dev/null 2>&1; then
            if docker compose -f "$compose_file" up -d kafka; then
                echo "Kafka container started (docker compose)."
            else
                echo "Failed to start Kafka via docker compose; ensure broker is running before Terraform apply." >&2
            fi
        elif command -v docker-compose >/dev/null 2>&1; then
            if docker-compose -f "$compose_file" up -d kafka; then
                echo "Kafka container started (docker-compose)."
            else
                echo "Failed to start Kafka via docker-compose; ensure broker is running before Terraform apply." >&2
            fi
        else
            echo "Docker Compose CLI not found; skipping Kafka startup." >&2
        fi
    else
        echo "Docker CLI not found; skipping Kafka startup." >&2
    fi
}

apply_terraform_dev() {
    if [ ! -d "$TERRAFORM_DIR" ]; then
        echo "Terraform directory not found: $TERRAFORM_DIR; skipping apply." >&2
        return
    fi

    echo "Applying Terraform configuration in $TERRAFORM_DIR..."
    (
        cd "$TERRAFORM_DIR"
        terraform init -input=false
        terraform apply -auto-approve -input=false
        if terraform output -raw topic_name >/dev/null 2>&1; then
            echo "Terraform output topic_name: $(terraform output -raw topic_name)"
        fi
    )
}

if [ "$INSTALL_NEEDED" = true ]; then
    ensure_kafka_running
    apply_terraform_dev
else
    echo "Terraform was already installed; skipping automatic apply."
fi
