-- Drop tables if they exist
DROP TABLE IF EXISTS tendermint_commit_signature;
DROP TABLE IF EXISTS tendermint_commit;
DROP TABLE IF EXISTS tendermint_status;
DROP TABLE IF EXISTS tendermint_peer_info;
DROP TABLE IF EXISTS tendermint_net_info;
DROP TABLE IF EXISTS tendermint_node_info;
DROP TABLE IF EXISTS alert_event_record;
DROP TABLE IF EXISTS active_alarm;
DROP TABLE IF EXISTS alarmer_level_association;
DROP TABLE IF EXISTS alert_level;
DROP TABLE IF EXISTS alarmer;
DROP TABLE IF EXISTS event;
DROP TABLE IF EXISTS agent_service;
DROP TABLE IF EXISTS service;
DROP TABLE IF EXISTS agent;
DROP TABLE IF EXISTS commit_record;
DROP TABLE IF EXISTS meta_monitor;

create table active_alarm
(
    alert_record_sent_time bigint       not null,
    alarmer_name           varchar(100) not null,
    target        varchar(100) not null,
    instance              varchar(100) not null,
    commit_id              varchar(255) not null,
    primary key (instance, target, alarmer_name)
);

create table agent_mark
(
    agent_name           varchar(100)                             not null,
    mark_start           datetime(6) default CURRENT_TIMESTAMP(6) not null,
    mark_end             datetime(6)                              null,
    marker_user_identity varchar(255)                             not null,
    marker_from          varchar(100)                             not null,
    primary key (agent_name, mark_start)
);

create table alert_event_record
(
    alert_record_uuid char(36)     not null
        primary key,
    start_timestamp   datetime     not null,
    resolv_timestamp  datetime     null,
    alert_name        varchar(100) not null,
    instance         varchar(100) not null,
    target   varchar(100) not null,
    commit_id         varchar(255) not null
);

# create table alert_record
# (
#     alert_record_uuid       char(36)     not null
#         primary key,
#     alert_record_created_at datetime(6)  not null,
#     alert_name              varchar(100) not null,
#     level_name              varchar(100) not null,
#     alarmer_name            varchar(255) not null,
#     agent_name              varchar(100) not null,
#     commit_id               varchar(255) not null
# );
#
# create index INDEX_created_at_alert_name_alarmer_function_name
#     on alert_record (alert_record_created_at, alert_name, alarmer_name);

create table commit_record
(
    commit_id  varchar(255) not null
        primary key,
    created_at datetime(6)  null
);

create table agent
(
    agent_name varchar(100) not null,
    commit_id  varchar(255) not null,
    host       varchar(30)  not null,
    port       int          null,
    platform   varchar(255) null,
    location   varchar(255) null,
    primary key (agent_name, commit_id),
    constraint FK_commit_record_TO_agent
        foreign key (commit_id) references commit_record (commit_id)
            on delete cascade
);


create table meta_monitor
(
    agent_name varchar(50) not null
        primary key,
    height     bigint      not null
);

create index INDEX_AGENT_NAME_HEIGHT
    on meta_monitor (agent_name asc, height desc);

create table service
(
    service_name  varchar(100) not null,
    commit_id     varchar(255) not null,
    monitor_image varchar(255) null,
    checker_image varchar(255) null,
    primary key (service_name, commit_id)
);

create table agent_service
(
    agent_name   varchar(100) not null,
    service_name varchar(100) not null,
    commit_id    varchar(255) not null,
    primary key (agent_name, service_name, commit_id),
    constraint FK_agent_TO_agent_service
        foreign key (agent_name, commit_id) references agent (agent_name, commit_id)
            on delete cascade,
    constraint FK_service_TO_agent_service
        foreign key (service_name, commit_id) references service (service_name, commit_id)
            on delete cascade
);

create table event
(
    event_uuid   char(36)     not null
        primary key,
    agent_name   varchar(100) not null,
    service_name varchar(100) not null,
    commit_id    varchar(255) not null,
    event_type   varchar(100) null,
    created_at   timestamp(6) null,
    constraint FK_agent_service_TO_event
        foreign key (agent_name, service_name, commit_id) references agent_service (agent_name, service_name, commit_id)
            on delete cascade
);

create table ethereum_block_number
(
    created_at   datetime(6) not null,
    event_uuid   char(36)    not null,
    block_number bigint      not null,
    primary key (created_at, event_uuid),
    constraint FK_event_TO_ethereum_block_number
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);

create index INDEX_commit_id__agent_name_service_name_created_at_event_uuid
    on event (commit_id asc, service_name asc, agent_name asc, created_at desc, event_uuid asc);

create table tendermint_commit
(
    created_at           datetime(6)  not null,
    event_uuid           char(36)     not null,
    chain_id             varchar(20)  null,
    height               bigint       null,
    time                 timestamp(6) null,
    last_block_id_hash   varchar(100) null,
    last_commit_hash     varchar(100) null,
    data_hash            varchar(100) null,
    validators_hash      varchar(100) null,
    next_validators_hash varchar(100) null,
    consensus_hash       varchar(100) null,
    app_hash             varchar(100) null,
    last_results_hash    varchar(100) null,
    evidence_hash        varchar(100) null,
    proposer_address     varchar(100) null,
    round                int          null,
    commit_block_id_hash varchar(100) null,
    primary key (created_at, event_uuid),
    constraint FK_event_TO_tendermint_commit
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);

create index INDEX_event_uuid_created_at_height
    on tendermint_commit (event_uuid asc, created_at desc, height asc);

create table tendermint_commit_signature
(
    validator_address            varchar(100) not null,
    tendermint_commit_created_at datetime(6)  not null,
    event_uuid                   char(36)     not null,
    timestamp                    timestamp(6) not null,
    signature                    varchar(200) not null,
    block_id_flag                int          not null,
    primary key (validator_address, tendermint_commit_created_at, event_uuid),
    constraint FK_tendermint_commit_TO_tendermint_commit_signature_list
        foreign key (event_uuid, tendermint_commit_created_at) references tendermint_commit (event_uuid, created_at)
            on delete cascade
);

create table tendermint_net_info
(
    created_at datetime(6) not null,
    event_uuid char(36)    not null,
    n_peers    int         null,
    listening  tinyint(1)  null,
    primary key (created_at, event_uuid),
    constraint FK_event_TO_tendermint_net_info
        foreign key (event_uuid) references event (event_uuid)
);

create table tendermint_node_info
(
    tendermint_node_info_uuid char(36)     not null
        primary key,
    node_id                   varchar(100) not null,
    listen_addr               varchar(255) not null,
    chain_id                  varchar(20)  not null,
    moniker                   varchar(50)  not null
);

create table tendermint_peer_info
(
    tendermint_peer_info_uuid char(36)    not null,
    created_at                datetime(6) not null,
    event_uuid                char(36)    not null,
    is_outbound               tinyint(1)  null,
    tendermint_node_info_uuid char(36)    not null,
    remote_ip                 varchar(50) null,
    primary key (tendermint_peer_info_uuid, created_at, event_uuid),
    constraint FK_tendermint_net_info_TO_tendermint_peer_info
        foreign key (event_uuid, created_at) references tendermint_net_info (event_uuid, created_at)
            on delete cascade,
    constraint FK_tendermint_node_info_TO_tendermint_peer_info
        foreign key (tendermint_node_info_uuid) references tendermint_node_info (tendermint_node_info_uuid)
            on delete cascade
);

create table tendermint_status
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    tendermint_node_info_uuid char(36)     not null,
    latest_block_hash         varchar(100) not null,
    latest_app_hash           varchar(100) not null,
    latest_block_height       bigint       not null,
    latest_block_time         timestamp(6) not null,
    earliest_block_hash       varchar(100) not null,
    earliest_app_hash         varchar(100) not null,
    earliest_block_height     bigint       not null,
    earliest_block_time       timestamp(6) not null,
    catching_up               tinyint(1)   not null,
    primary key (created_at, event_uuid),
    constraint FK_event_TO_tendermint_status
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade,
    constraint FK_tendermint_node_info_TO_tendermint_status
        foreign key (tendermint_node_info_uuid) references tendermint_node_info (tendermint_node_info_uuid)
            on delete cascade
);


# suffix of `status` means the single metric is able to judge.
# suffix of `total` means the needed to combine with before one.
create table node_free_disk_status
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    mountpoint varchar(100) not null,

    device varchar(100) not null,
    free_disk_size bigint not null,
    primary key (event_uuid, created_at, mountpoint), # to filter by mountpoint without accessing table, `mountpoint` is added to PK for indexing.
    constraint FK_event_TO_node_free_disk_status
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);

create table node_cpu_seconds_total
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    mode varchar(30) not null,

    number int not null, # cpu number
    seconds_total bigint not null,
    primary key (event_uuid, created_at, mode), # to filter by mode without accessing table. `mode` is added to PK for indexing.
    constraint FK_event_TO_node_cpu_seconds_total
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);

create table node_memory_status
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    total bigint not null,
    free bigint not null,
    buffer bigint not null,
    cached bigint not null,
    slab bigint not null,
    page_tables bigint not null,
    swap_cached bigint not null,

    primary key (event_uuid, created_at),
    constraint FK_event_TO_node_memory_status
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);


create table node_systemd_status
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    name varchar(100) not null,
    state varchar(50) not null, # == each node_systemd_unit_state will return only one state, and it'll be used to this field.

    primary key (event_uuid, created_at, name, state),
    constraint FK_event_TO_node_systemd_status
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);


create table node_network_total
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    device varchar(100) not null,

    receive_total bigint not null,
    transmit_total bigint not null,

    primary key (event_uuid, created_at, device),
    constraint FK_event_TO_node_network_total
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);


create table hyperliquid_status
(
    created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,
    timestamp datetime(6) not null,

    round bigint not null,
    home_validator varchar(100) not null,

    primary key (event_uuid, created_at, timestamp),
    constraint FK_event_TO_hyperliquid_status
        foreign key (event_uuid) references event (event_uuid)
            on delete cascade
);


create table hyperliquid_status_validator
(
    hyperliquid_status_created_at                datetime(6)  not null,
    event_uuid                char(36)     not null,


    validator_address varchar(100) not null,
    stakes bigint not null,
    is_jailed boolean not null,
    is_next boolean not null,
    is_missing_heartbeat boolean not null,

    since_last_success float not null,
    last_ack_duration float not null,

    primary key (event_uuid, hyperliquid_status_created_at, validator_address),
    constraint FK_hyperliquid_status_TO_hyperliquid_status_validator
        foreign key (event_uuid, hyperliquid_status_created_at) references hyperliquid_status (event_uuid, created_at)
            on delete cascade
);

