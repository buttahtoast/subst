
FROM bash:5
ENV ARGOCD_EXEC_TIMEOUT=90s
COPY subst /subst
COPY argocd-cmp/cmp.yaml /home/argocd/cmp-server/config/plugin.yaml
COPY argocd-cmp/entrypoint.sh /entrypoint.sh
RUN adduser -H -D -s /bin/bash -G nobody -u 999 argocd && \
    chmod +x /entrypoint.sh
USER argocd:nobody
ENTRYPOINT ["/entrypoint.sh"]

