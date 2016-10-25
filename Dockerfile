FROM alpine:3.2

RUN echo "http://dl-4.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && apk update && apk add python wget ca-certificates curl py-m2crypto

COPY ./docker/requests-2.5.1.tar.gz /tmp/requests.tar

RUN tar xzvf /tmp/requests.tar && cd /requests-2.5.1 && python setup.py install && rm /tmp/requests.tar && rm -r /requests-2.5.1

COPY ./vault-linux-amd64 /usr/bin/vault
RUN chmod +x /usr/bin/vault
COPY ./docker/vault.conf.tmp /etc/vault.conf.tmp
COPY ./docker/dev.conf /etc/dev.conf
COPY ./docker/start.sh /start.sh
COPY ./docker/vault_check.py /vault_check.py
COPY ./docker/ca-serve /usr/bin/ca-serve


RUN mkdir -p /etc/ca-certs/root && mkdir -p /etc/ca-certs/intermediate && cp /etc/certs/cachain.crt /etc/ca-certs/root/cachain.crt

RUN chmod +x /start.sh; chmod +x /usr/bin/vault; chmod +x /usr/bin/ca-serve

ENV ZK_HOSTS=127.0.0.1:2181 TLS_DISABLE=false TLS_CERT=/etc/certs/default.crt TLS_KEY=/etc/certs/default.key HOST=127.0.0.1 DEV=true TLS_CA=/etc/certs/cachain.crt BACKEND=zookeeper CHIPPER_URL=http://chipper.bl2.yammer.com DISCOVER_ZK=false

EXPOSE 8200
CMD /start.sh
