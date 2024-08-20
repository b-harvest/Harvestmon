
insert into commit_record(commit_id, created_at) values ('0f746d26a5a8e6c2d6cddac88a6dad4582fa4eac', now());

insert into agent(agent_name, commit_id, host, platform, location) values('[TV]axelar-tv-ovh', '0f746d26a5a8e6c2d6cddac88a6dad4582fa4eac', '51.75.145.103', 'ovh', 'ff');
insert into service(service_name, commit_id, monitor_image, checker_image) values('tendermint', '0f746d26a5a8e6c2d6cddac88a6dad4582fa4eac', 'public.ecr.aws/j7u6r2t4/harvestmon-tendermint:v0.0.7', 'https://ap-northeast-2.console.aws.amazon.com/ecr/repositories/harvestmon-checker-tendermint?region=ap-northeast-2');
insert into agent_service(agent_name, service_name, commit_id) values ('[TV]axelar-tv-ovh', 'tendermint', '0f746d26a5a8e6c2d6cddac88a6dad4582fa4eac');
