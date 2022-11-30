CREATE TABLE migration
(
    id SERIAL PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    created_at BIGINT NOT NULL
);


CREATE TABLE transaction
(
	transaction_hash TEXT PRIMARY KEY NOT NULL,
	transaction_status INT NOT NULL,
	block_hash TEXT NOT NULL,
    block_number INT NOT NULL,
	from_address TEXT NOT NULL,
	to_address TEXT,
	contract_address TEXT,
	logs_count INT NOT NULL,
	input TEXT NOT NULL,
	value BIGINT NOT NULL
);

CREATE TABLE token_transaction
(
    token TEXT NOT NULL,
    transaction_hash TEXT NOT NULL REFERENCES transaction (transaction_hash),
    PRIMARY KEY (token, transaction_hash)
);

CREATE TABLE users
(
    username TEXT NOT NULL PRIMARY KEY,
    password TEXT NOT NULL
);


INSERT INTO users (username, password)
VALUES 
    ('alice', 'alice'),
    ('bob', 'bob'),
    ('carol', 'carol'),
    ('dave', 'dave');

CREATE TABLE user_token
(
    id SERIAL PRIMARY KEY NOT NULL,
    username TEXT NOT NULL REFERENCES users (username),
    token TEXT NOT NULL
);

CREATE INDEX user_token_username_index ON user_token (username);