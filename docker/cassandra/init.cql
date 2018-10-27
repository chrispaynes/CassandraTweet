CREATE KEYSPACE alpha WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};
CREATE TABLE alpha.tweet(id UUID, created_at ascii, text text, PRIMARY KEY(id));