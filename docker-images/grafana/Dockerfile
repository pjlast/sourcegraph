# sourcegraph/grafana - learn more about this image in https://docs.sourcegraph.com/dev/background-information/observability/grafana

# Build monitoring definitions
FROM sourcegraph/alpine-3.14:180512_2022-10-31_84d1e240bb40@sha256:179ad53ab463ebc804f93de967113739fa73efc2cea6d9c53a9106be45f79d5e AS monitoring_builder
RUN mkdir -p '/generated/grafana'
COPY ./.bin/monitoring-generator /bin/monitoring-generator
RUN GRAFANA_DIR='/generated/grafana' PROMETHEUS_DIR='' DOCS_DIR='' NO_PRUNE=true /bin/monitoring-generator
RUN ls '/generated/grafana'

# Prepare final image
# when upgrading the Grafana version, please refer to https://docs.sourcegraph.com/dev/background-information/observability/grafana#upgrading-grafana
# DO NOT UPGRADE to AGPL Grafana without consulting Stephen+legal, Grafana >= 8.0 is AGPLv3 Licensed
# See https://docs.google.com/document/d/1nSmz1ChL_rBvX8FAKTB-CNzgcff083sUlIpoXEz6FHE/edit#heading=h.69clsrno4211
FROM grafana/grafana:7.5.17@sha256:15abb652aa82eeb9f45589278b34ae6ef0e96f74c389cadde31831eb0b1ce228 as production
LABEL com.sourcegraph.grafana.version=7.5.17

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.url=https://sourcegraph.com/
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/
LABEL org.opencontainers.image.documentation=https://docs.sourcegraph.com/

# hadolint ignore=DL3020
ADD config /sg_config_grafana
COPY --from=monitoring_builder /generated/grafana/home.json /usr/share/grafana/public/dashboards/home.json
COPY --from=monitoring_builder /generated/grafana/* /sg_config_grafana/provisioning/dashboards/sourcegraph/

# hadolint ignore=DL3020
ADD entry.sh /


USER root

# Create optional folders to avoid error logs about missing dirs
RUN mkdir /sg_grafana_additional_dashboards
RUN mkdir /sg_config_grafana/provisioning/plugins && chown grafana:root /sg_config_grafana/provisioning/plugins

# @FIXME: Update redis image
# Pin busybox=1.32.1-r7 https://github.com/sourcegraph/sourcegraph/issues/27965
RUN apk add --upgrade --no-cache \
    'apk-tools>=2.12' \
    'krb5-libs>=1.18.4' \
    'libssl1.1>=1.1.1l' \
    'openssl>=1.1.1l' \
    'busybox>=1.32.1' \
    'ncurses-libs>=6.2_p20210109-r1' \
    'ncurses-terminfo-base>=6.2_p20210109-r1' \
    'libtirpc>=1.3.1-r1'

EXPOSE 3370
USER grafana
ENTRYPOINT ["/entry.sh"]
