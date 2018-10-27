.PHONY: cassandraInit

cassandraInit:
	docker-compose exec cassandra bash -c "cqlsh --file='./init.cql'"
