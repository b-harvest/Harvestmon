
insert into commit_record(commit_id, created_at) values ('0f746d26a5a8e6c2d6cddac88a6dad4582fa4eac', now())

create user hello;
create database harvestmon;

SET PASSWORD FOR 'hello' = PASSWORD('helloworld');

insert into commit_record(commit_id, created_at) value ('19ge4rgndfifji', now());
insert into agent(agent_name, commit_id, ipv4_address, platform, location) values('polkachu.com', '19ge4rgndfifji', 'cosmos-rpc.polkachu.com', null, null);
insert into service(service_name, commit_id, monitor_image, checker_image) values('tendermint-mon', '19ge4rgndfifji', 'ghcr.io/b-harvest/tendermint-mon', 'ghcr.io/b-harvest/tendermint-checker');
insert into agent_service(agent_name, service_name, commit_id) values ('polkachu.com', 'tendermint-mon', '19ge4rgndfifji');
