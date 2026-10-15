ARG OMNIGREX_GOLDEN_IMAGE
FROM ${OMNIGREX_GOLDEN_IMAGE}

LABEL org.opencontainers.image.source="https://github.com/jozala/omnigrex-testrun"

USER root

# The current Omnigrex Pi golden image is Debian Bookworm based.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg \
         -o /etc/apt/keyrings/docker.asc \
    && chmod a+r /etc/apt/keyrings/docker.asc \
    && printf '%s\n' \
         "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable" \
         > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
         docker-ce \
         docker-ce-cli \
         containerd.io \
         docker-buildx-plugin \
         docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

USER 10001:10001
WORKDIR /workspace