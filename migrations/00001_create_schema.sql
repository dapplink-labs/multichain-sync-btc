DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
            CREATE DOMAIN UINT256 AS NUMERIC
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        ELSE
            ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
            ALTER DOMAIN UINT256 ADD
                CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        END IF;
    END
$$;

CREATE TABLE IF NOT EXISTS business
(
    guid          VARCHAR PRIMARY KEY,
    business_uid  VARCHAR NOT NULL,
    notify_url    VARCHAR NOT NULL,
    call_back_url VARCHAR NOT NULL,
    timestamp     INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS tokens_timestamp ON business (timestamp);
CREATE INDEX IF NOT EXISTS business_uid ON business (business_uid);

CREATE TABLE IF NOT EXISTS blocks
(
    hash        VARCHAR PRIMARY KEY,
    prev_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE CHECK (number > 0),
    timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS blocks_number ON blocks (number);
CREATE INDEX IF NOT EXISTS blocks_timestamp ON blocks (timestamp);


CREATE TABLE IF NOT EXISTS reorg_blocks
(
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE CHECK (number > 0),
    timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS reorg_blocks_number ON reorg_blocks (number);
CREATE INDEX IF NOT EXISTS reorg_blocks_timestamp ON reorg_blocks (timestamp);


CREATE TABLE IF NOT EXISTS addresses
(
    guid         VARCHAR PRIMARY KEY,
    address      VARCHAR UNIQUE NOT NULL,
    address_type SMALLINT       NOT NULL DEFAULT 0,
    public_key   VARCHAR        NOT NULL,
    timestamp    INTEGER        NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS addresses_address ON addresses (address);
CREATE INDEX IF NOT EXISTS addresses_timestamp ON addresses (timestamp);

CREATE TABLE IF NOT EXISTS balances
(
    guid          VARCHAR PRIMARY KEY,
    address       VARCHAR  NOT NULL,
    address_type  SMALLINT NOT NULL DEFAULT 0,
    balance       UINT256  NOT NULL CHECK (balance >= 0),
    lock_balance  UINT256  NOT NULL,
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS balances_address ON balances (address);
CREATE INDEX IF NOT EXISTS balances_timestamp ON balances (timestamp);

CREATE TABLE IF NOT EXISTS vins
(
    guid               VARCHAR PRIMARY KEY,
    address            VARCHAR  NOT NULL,
    txid               VARCHAR  NOT NULL,
    vout               SMALLINT NOT NULL DEFAULT 0,
    script             VARCHAR,
    witness            VARCHAR,
    amount             UINT256  NOT NULL CHECK (amount >= 0),
    spend_tx_hash      VARCHAR NOT NULL,
    spend_block_height UINT256  NOT NULL CHECK (spend_block_height >= 0),
    is_spend           BOOL DEFAULT FALSE,
    timestamp          INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS vins_address ON vins(address);
CREATE INDEX IF NOT EXISTS vins_timestamp ON vins (timestamp);


CREATE TABLE IF NOT EXISTS vouts
(
    guid          VARCHAR PRIMARY KEY,
    address       VARCHAR  NOT NULL,
    n             SMALLINT NOT NULL DEFAULT 0,
    script        VARCHAR,
    amount        UINT256  NOT NULL CHECK (amount >= 0),
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS vouts_address ON vouts(address);
CREATE INDEX IF NOT EXISTS vouts_timestamp ON vouts(timestamp);

CREATE TABLE IF NOT EXISTS deposits
(
    guid          VARCHAR PRIMARY KEY,
    block_hash    VARCHAR  NOT NULL,
    block_number  UINT256  NOT NULL CHECK (block_number > 0),
    hash          VARCHAR  NOT NULL,
    fee           UINT256  NOT NULL,
    lock_time     UINT256  NOT NULL,
    version       VARCHAR  NOT NULL,
    confirms      SMALLINT NOT NULL DEFAULT 0,
    status        SMALLINT NOT NULL DEFAULT 0,
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS deposits_hash ON deposits (hash);
CREATE INDEX IF NOT EXISTS deposits_timestamp ON deposits (timestamp);

CREATE TABLE IF NOT EXISTS withdraws
(
    guid                     VARCHAR PRIMARY KEY,
    block_hash               VARCHAR  NOT NULL,
    block_number             UINT256  NOT NULL CHECK (block_number > 0),
    hash                     VARCHAR  NOT NULL,
    fee                      VARCHAR  NOT NULL,
    lock_time                UINT256  NOT NULL,
    version                  VARCHAR  NOT NULL,
    tx_sign_hex              VARCHAR  NOT NULL,
    status                   SMALLINT NOT NULL DEFAULT 0,
    timestamp                INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS withdraws_hash ON withdraws (hash);
CREATE INDEX IF NOT EXISTS withdraws_timestamp ON withdraws (timestamp);

CREATE TABLE IF NOT EXISTS internals
(
    guid                     VARCHAR PRIMARY KEY,
    status                   SMALLINT NOT NULL DEFAULT 0,
    block_hash               VARCHAR  NOT NULL,
    block_number             UINT256  NOT NULL CHECK (block_number > 0),
    hash                     VARCHAR  NOT NULL,
    fee                      VARCHAR  NOT NULL,
    lock_time                UINT256  NOT NULL,
    version                  VARCHAR  NOT NULL,
    tx_sign_hex              VARCHAR  NOT NULL,
    timestamp                INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS internals_hash ON internals (hash);
CREATE INDEX IF NOT EXISTS internals_timestamp ON internals (timestamp);


CREATE TABLE IF NOT EXISTS transactions
(
    guid          VARCHAR PRIMARY KEY,
    block_hash    VARCHAR  NOT NULL,
    block_number  UINT256  NOT NULL CHECK (block_number > 0),
    hash          VARCHAR  NOT NULL,
    fee           UINT256  NOT NULL,
    lock_time     UINT256  NOT NULL,
    version       VARCHAR  NOT NULL,
    status        SMALLINT NOT NULL DEFAULT 0,
    tx_type       VARCHAR  NOT NULL,
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS transactions_hash ON transactions (hash);
CREATE INDEX IF NOT EXISTS transactions_timestamp ON transactions (timestamp);


CREATE TABLE IF NOT EXISTS child_txs (
    guid          VARCHAR PRIMARY KEY,
    hash          VARCHAR  NOT NULL,
    tx_id          VARCHAR  NOT NULL,
    tx_index      UINT256  NOT NULL,
    from_address  VARCHAR  NOT NULL,
    to_address    VARCHAR  NOT NULL,
    amount        VARCHAR  NOT NULL,
    tx_type       VARCHAR  NOT NULL,
    timestamp     INTEGER  NOT NULL CHECK (timestamp > 0)
)
CREATE INDEX IF NOT EXISTS child_txs_tx_hash ON child_txs (hash);
CREATE INDEX IF NOT EXISTS child_txs_timestamp ON child_txs (timestamp);

