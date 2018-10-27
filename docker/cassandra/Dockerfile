FROM cassandra:2.1.20

RUN apt-get update

COPY ./docker/cassandra/entrypoint-init.sh .

RUN chmod +x entrypoint-init.sh \
        && sh ./entrypoint-init.sh

CMD ["cassandra", "-f"]
