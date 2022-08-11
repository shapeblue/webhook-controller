FROM ubuntu:20.04

ARG KUBERNETES_VERSION
ENV KUBERNETES_VERSION ${KUBERNETES_VERSION}
# Install Docker
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections 
#RUN DEBIAN_FRONTEND="noninteractive" TZ="America/New_York" apt-get update -y && \
RUN apt-get update -y && \
    apt install -y apt-transport-https ca-certificates curl software-properties-common wget jq make build-essential && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable" && \
    apt install -y docker-ce

# Install cmk
RUN wget https://github.com/apache/cloudstack-cloudmonkey/releases/download/6.2.0/cmk.linux.x86-64 -O /usr/local/bin/cmk && \
    chmod +x /usr/local/bin/cmk

# Install go
RUN rm -rf /usr/local/go && \
    wget https://go.dev/dl/go1.17.12.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.17.12.linux-amd64.tar.gz && \
    rm -rf go1.17.12.linux-amd64.tar.gz

# Export go path
RUN export PATH=$PATH:/usr/local/go/bin && export GOPATH="/root/go/" && export GOBIN="/usr/local/go/bin"

# Install kind
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.14.0/kind-linux-amd64 && \
    chmod +x ./kind && mv ./kind /usr/local/bin/kind

# Install kubectl - k8s version exported as env variable via jenkins cloud agent config
# And validate the binary
RUN curl -LO "https://dl.k8s.io/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl" && \
    curl -LO "https://dl.k8s.io/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl.sha256" && \
    echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    
# Install clusterctl
RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.2.0/clusterctl-linux-amd64 -o clusterctl && \
    chmod +x ./clusterctl && \
    mv ./clusterctl /usr/local/bin/clusterctl
