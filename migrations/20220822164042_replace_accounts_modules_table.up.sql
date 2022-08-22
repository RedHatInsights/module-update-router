CREATE TABLE orgs_modules (
    module_name VARCHAR(256),
    org_id VARCHAR(256),
    PRIMARY KEY(module_name, org_id)
);

DROP TABLE IF EXISTS accounts_modules;
