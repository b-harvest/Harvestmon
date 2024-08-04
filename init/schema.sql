-- Drop tables if they exist
DROP TABLE IF EXISTS tendermint_commit_signature_list;
DROP TABLE IF EXISTS tendermint_commit;
DROP TABLE IF EXISTS tendermint_status;
DROP TABLE IF EXISTS tendermint_peer_info;
DROP TABLE IF EXISTS tendermint_net_info;
DROP TABLE IF EXISTS tendermint_node_info;
DROP TABLE IF EXISTS alert_record;
DROP TABLE IF EXISTS alarmer_level_association;
DROP TABLE IF EXISTS alarmer_env;
DROP TABLE IF EXISTS alert_level;
DROP TABLE IF EXISTS alarmer;
DROP TABLE IF EXISTS event;
DROP TABLE IF EXISTS agent_service;
DROP TABLE IF EXISTS service;
DROP TABLE IF EXISTS agent;
DROP TABLE IF EXISTS commit_record;

-- Create tables
CREATE TABLE `agent_service` (
    `agent_name`	varchar(100)	NOT NULL,
    `service_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL
);

CREATE TABLE `agent` (
    `agent_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL,

    `host`	varchar(30)	NOT NULL,
    `port` int NULL,
    `platform`	varchar(255)	NULL,
    `location`	varchar(255)	NULL
);

CREATE TABLE `service` (
    `service_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL,

    `monitor_image`	varchar(255)	NULL,
    `checker_image`	varchar(255)	NULL
);

CREATE TABLE `event` (
    `event_uuid`	varchar(255)	NOT NULL,

    `agent_name`	varchar(100)	NOT NULL,
    `service_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL,

    `event_type`	varchar(100)	NULL,
    `created_at`	timestamp(6)	NULL
);

CREATE TABLE `commit_record` (
    `commit_id`	varchar(255)	NOT NULL,
    `created_at`	datetime(6)	NULL
);

CREATE TABLE `alert_level` (
    `level_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL
);

CREATE TABLE `tendermint_status` (
    `created_at`	datetime(6)	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,

    `tendermint_node_info_uuid`	UUID	NOT NULL,

    `latest_block_hash`	varchar(100)	NULL,
    `latest_app_hash`	varchar(100)	NULL,
    `latest_block_height`	BigInt	NULL,
    `latest_block_time`	timestamp(6)	NULL,
    `earliest_block_hash`	varchar(100)	NULL,
    `earliest_app_hash`	varchar(100)	NULL,
    `earliest_block_height`	BigInt	NULL,
    `earliest_block_time`	timestamp(6)	NULL,
    `catching_up`	Bool	NULL
);

CREATE TABLE `tendermint_net_info` (
    `created_at`	datetime(6)	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,

    `n_peers`	Int	NULL,
    `listening`	Bool	NULL
);

CREATE TABLE `tendermint_commit` (
    `created_at`	datetime(6)	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,

    `chain_id`	varchar(20)	NULL,
    `height`	BigInt	NULL,
    `time`	timestamp(6)	NULL,
    `last_block_id_hash`	varchar(100)	NULL,
    `last_commit_hash`	varchar(100)	NULL,
    `data_hash`	varchar(100)	NULL,
    `validators_hash`	varchar(100)	NULL,
    `next_validators_hash`	varchar(100)	NULL,
    `consensus_hash`	varchar(100)	NULL,
    `app_hash`	varchar(100)	NULL,
    `last_results_hash`	varchar(100)	NULL,
    `evidence_hash`	varchar(100)	NULL,
    `proposer_address`	varchar(100)	NULL,
    `round`	Int	NULL,
    `commit_block_id_hash`	varchar(100)	NULL
);

CREATE TABLE `tendermint_node_info` (
    `tendermint_node_info_uuid`	UUID	NOT NULL,

    `node_id`	varchar(100)	NULL,
    `listen_addr`	varchar(255)	NULL,
    `chain_id`	varchar(20)	NULL,
    `moniker`	varchar(50)	NULL
);

CREATE TABLE `tendermint_peer_info` (
    `tendermint_peer_info_uuid`	UUID	NOT NULL,

    `created_at`	datetime(6)	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,

    `is_outbound`	Bool	NULL,
    `tendermint_node_info_uuid`	UUID	NOT NULL,
    `remote_ip`	varchar(50)	NULL
);

CREATE TABLE `tendermint_commit_signature_list` (
    `validator_address`	varchar(100)	NOT NULL,
    `created_at`	datetime(6)	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,

    `timestamp`	timestamp(6)	NOT NULL,
    `signature`	varchar(200)	NOT NULL,
    `block_id_flag`	Int	NOT NULL
);

CREATE TABLE `alarmer` (
    `alarmer_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL,

    `image`	varchar(255)	NULL
);

CREATE TABLE `alarmer_level_association` (
    `level_name`	varchar(100)	NOT NULL,
    `alarmer_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL
);

CREATE TABLE `alarmer_env` (
    `env_name`	varchar(255)	NOT NULL,
    `alarmer_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL,

    `env_value`	varchar(500)	NULL
);

CREATE TABLE `alert_record` (
    `alert_uuid`	UUID	NOT NULL,
    `event_uuid`	varchar(255)	NOT NULL,
    `level_name`	varchar(100)	NOT NULL,
    `commit_id`	varchar(255)	NOT NULL
);


-- Create PK on tables.
ALTER TABLE `agent_service` ADD CONSTRAINT `PK_AGENT_SERVICE` PRIMARY KEY (
    `agent_name`,
    `service_name`,
    `commit_id`
);

ALTER TABLE `agent` ADD CONSTRAINT `PK_AGENT` PRIMARY KEY (
    `agent_name`,
    `commit_id`
);

ALTER TABLE `service` ADD CONSTRAINT `PK_SERVICE` PRIMARY KEY (
    `service_name`,
    `commit_id`
);

ALTER TABLE `event` ADD CONSTRAINT `PK_EVENT` PRIMARY KEY (
    `event_uuid`
);

ALTER TABLE `commit_record` ADD CONSTRAINT `PK_COMMIT_RECORD` PRIMARY KEY (
    `commit_id`
);

ALTER TABLE `alert_level` ADD CONSTRAINT `PK_ALERT_LEVEL` PRIMARY KEY (
    `level_name`,
    `commit_id`
);

ALTER TABLE `tendermint_status` ADD CONSTRAINT `PK_TENDERMINT_STATUS` PRIMARY KEY (
    `created_at`,
    `event_uuid`
);

ALTER TABLE `tendermint_net_info` ADD CONSTRAINT `PK_TENDERMINT_NET_INFO` PRIMARY KEY (
    `created_at`,
    `event_uuid`
);

ALTER TABLE `tendermint_commit` ADD CONSTRAINT `PK_TENDERMINT_COMMIT` PRIMARY KEY (
    `created_at`,
    `event_uuid`
);

ALTER TABLE `tendermint_node_info` ADD CONSTRAINT `PK_TENDERMINT_NODE_INFO` PRIMARY KEY (
    `tendermint_node_info_uuid`
);

ALTER TABLE `tendermint_peer_info` ADD CONSTRAINT `PK_TENDERMINT_PEER_INFO` PRIMARY KEY (
    `tendermint_peer_info_uuid`,
    `created_at`,
    `event_uuid`
);

ALTER TABLE `tendermint_commit_signature_list` ADD CONSTRAINT `PK_TENDERMINT_COMMIT_SIGNATURE_LIST` PRIMARY KEY (
    `validator_address`,
    `created_at`,
    `event_uuid`
);

ALTER TABLE `alarmer` ADD CONSTRAINT `PK_ALARMER` PRIMARY KEY (
    `alarmer_name`,
    `commit_id`
);

ALTER TABLE `alarmer_level_association` ADD CONSTRAINT `PK_ALARMER_LEVEL_ASSOCIATION` PRIMARY KEY (
    `level_name`,
    `alarmer_name`,
    `commit_id`
);

ALTER TABLE `alarmer_env` ADD CONSTRAINT `PK_ALARMER_ENV` PRIMARY KEY (
    `env_name`,
    `alarmer_name`,
    `commit_id`
);

ALTER TABLE `alert_record` ADD CONSTRAINT `PK_ALERT_RECORD` PRIMARY KEY (
    `alert_uuid`,
    `event_uuid`,
    `level_name`,
    `commit_id`
);

# FK constraints
ALTER TABLE `agent_service` ADD CONSTRAINT `FK_agent_TO_agent_service_1` FOREIGN KEY (`agent_name`,`commit_id`)
REFERENCES `agent` (`agent_name`,`commit_id`);

ALTER TABLE `agent_service` ADD CONSTRAINT `FK_service_TO_agent_service_1` FOREIGN KEY (`service_name`,`commit_id`)
REFERENCES `service` (`service_name`,`commit_id`);

ALTER TABLE `agent` ADD CONSTRAINT `FK_commit_record_TO_agent_1` FOREIGN KEY (`commit_id`)
REFERENCES `commit_record` (`commit_id`);

ALTER TABLE `event` ADD CONSTRAINT `FK_agent_service_TO_event_1` FOREIGN KEY (`agent_name`, `service_name`, `commit_id`)
REFERENCES `agent_service` (`agent_name`, `service_name`, `commit_id`);

ALTER TABLE `tendermint_status` ADD CONSTRAINT `FK_event_TO_tendermint_status_1` FOREIGN KEY (`event_uuid`)
REFERENCES `event` (`event_uuid`);

ALTER TABLE `tendermint_status` ADD CONSTRAINT `FK_tendermint_node_info_TO_tendermint_status_1` FOREIGN KEY (`tendermint_node_info_uuid`)
REFERENCES `tendermint_node_info` (`tendermint_node_info_uuid`);

ALTER TABLE `tendermint_peer_info` ADD CONSTRAINT `FK_tendermint_node_info_TO_tendermint_peer_info_1` FOREIGN KEY (`tendermint_node_info_uuid`)
REFERENCES `tendermint_node_info` (`tendermint_node_info_uuid`);

ALTER TABLE `tendermint_net_info` ADD CONSTRAINT `FK_event_TO_tendermint_net_info_1` FOREIGN KEY (`event_uuid`)
REFERENCES `event` (`event_uuid`);

ALTER TABLE `tendermint_commit` ADD CONSTRAINT `FK_event_TO_tendermint_commit_1` FOREIGN KEY (`event_uuid`)
REFERENCES `event` (`event_uuid`);

ALTER TABLE `tendermint_peer_info` ADD CONSTRAINT `FK_tendermint_net_info_TO_tendermint_peer_info_1` FOREIGN KEY (`event_uuid`, `created_at`)
REFERENCES `tendermint_net_info` (`event_uuid`, `created_at`);

ALTER TABLE `tendermint_peer_info` ADD CONSTRAINT `FK_event_TO_tendermint_peer_info_1` FOREIGN KEY (`event_uuid`)
REFERENCES `tendermint_net_info` (`event_uuid`);

ALTER TABLE `tendermint_commit_signature_list` ADD CONSTRAINT `FK_tendermint_commit_TO_tendermint_commit_signature_list_1` FOREIGN KEY (`event_uuid`, `created_at`)
REFERENCES `tendermint_commit` (`event_uuid`, `created_at`);

ALTER TABLE `tendermint_commit_signature_list` ADD CONSTRAINT `FK_event_TO_tendermint_commit_signature_list_1` FOREIGN KEY (`event_uuid`)
REFERENCES `tendermint_commit` (`event_uuid`);

ALTER TABLE `alert_level` ADD CONSTRAINT `FK_commit_record_TO_alert_level_1` FOREIGN KEY (`commit_id`)
REFERENCES `commit_record` (`commit_id`);

ALTER TABLE `alarmer` ADD CONSTRAINT `FK_commit_record_TO_alarmer_1` FOREIGN KEY (`commit_id`)
REFERENCES `commit_record` (`commit_id`);

ALTER TABLE `alarmer_level_association` ADD CONSTRAINT `FK_alert_level_TO_alarmer_level_association_1` FOREIGN KEY (`level_name`, `commit_id`)
REFERENCES `alert_level` (`level_name`, `commit_id`);

ALTER TABLE `alarmer_level_association` ADD CONSTRAINT `FK_alarmer_TO_alarmer_level_association_1` FOREIGN KEY (`alarmer_name`, `commit_id`)
REFERENCES `alarmer` (`alarmer_name`, `commit_id`);

ALTER TABLE `alarmer_env` ADD CONSTRAINT `FK_alarmer_TO_alarmer_env_1` FOREIGN KEY (`alarmer_name`, `commit_id`)
REFERENCES `alarmer` (`alarmer_name`, `commit_id`);

ALTER TABLE `alert_record` ADD CONSTRAINT `FK_event_TO_alert_record_1` FOREIGN KEY (`event_uuid`)
REFERENCES `event` (`event_uuid`);

ALTER TABLE `alert_record` ADD CONSTRAINT `FK_alert_level_TO_alert_record_1` FOREIGN KEY (`level_name`, `commit_id`)
REFERENCES `alert_level` (`level_name`, `commit_id`);

