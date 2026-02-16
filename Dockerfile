FROM oven/bun:1-debian

RUN apt-get update && apt-get install -y \
    curl \
    git \
    ca-certificates \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install opencode and pi at build time
RUN bun install -g opencode-ai @mariozechner/pi-coding-agent

ENV PATH="/root/.bun/install/global/bin:/root/.bun/bin:$PATH"

WORKDIR /workspace

CMD ["/bin/bash"]
