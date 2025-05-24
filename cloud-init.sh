#!/bin/bash

# Update the system
dnf update -y

# Install prerequisites
dnf install -y dnf-utils zip unzip wget

# Install Docker
dnf config-manager --add-repo=https://download.docker.com/linux/centos/docker-ce.repo
dnf install -y docker-ce docker-ce-cli containerd.io
systemctl enable docker
systemctl start docker

# Install Docker Compose (standalone binary for ARM64)
COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-aarch64" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose

# Add user to docker group to run Docker without sudo
usermod -aG docker opc

# Install Python
dnf install -y python3 python3-pip
# Ensure pip is upgraded
pip3 install --upgrade pip

# Install Go (latest version for ARM64)
GO_VERSION=$(curl -s https://golang.org/dl/ | grep -oP 'go[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)
curl -L "https://golang.org/dl/${GO_VERSION}.linux-arm64.tar.gz" -o /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

# Set up Go environment variables
echo 'export PATH=$PATH:/usr/local/go/bin' >> /home/opc/.bashrc
echo 'export GOPATH=/home/opc/go' >> /home/opc/.bashrc
mkdir -p /home/opc/go/{bin,src,pkg}
chown -R opc:opc /home/opc/go

# Verify installations
docker --version
docker-compose --version
python3 --version
pip3 --version
/usr/local/go/bin/go version

# Log versions to a file for verification
echo "Installed versions:" > /home/opc/install_versions.txt
docker --version >> /home/opc/install_versions.txt
docker-compose --version >> /home/opc/install_versions.txt
python3 --version >> /home/opc/install_versions.txt
pip3 --version >> /home/opc/install_versions.txt
/usr/local/go/bin/go version >> /home/opc/install_versions.txt

# Ensure opc user has proper permissions
chown opc:opc /home/opc/install_versions.txt