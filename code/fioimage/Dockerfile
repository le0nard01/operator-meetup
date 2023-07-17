FROM fedora:38

ENV readThreshold=0 \
    writeThreshold=0

RUN dnf install fio jq -y && dnf clean all -y

COPY script.sh .
RUN chmod +x script.sh
ENTRYPOINT sh script.sh 