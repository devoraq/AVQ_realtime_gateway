#!/bin/sh
set -eu

if command -v terraform >/dev/null 2>&1; then
    echo "Terraform already installed: $(terraform version | head -n 1)"
    exit 0
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
        install_debian_like
        ;;
    arch|manjaro|endeavouros|arco|archarm)
        install_arch_like
        ;;
    *)
        if echo "${ID_LIKE:-}" | grep -Eq 'debian|ubuntu'; then
            install_debian_like
        elif echo "${ID_LIKE:-}" | grep -Eq 'arch'; then
            install_arch_like
        else
            echo "This script supports only Debian/Ubuntu and Arch-based systems." >&2
            exit 1
        fi
        ;;
esac

echo "Terraform installed:"
terraform version | head -n 1
