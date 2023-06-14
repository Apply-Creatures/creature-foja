FROM public.ecr.aws/docker/library/alpine:3.18
RUN mkdir -p /app/gitea
RUN ( echo '#!/bin/sh' ; echo "echo forgejo v1.2.3" ) > /app/gitea/gitea ; chmod +x /app/gitea/gitea
