-- see generate.go

-- DROP DATABASE marvin_test;
-- DROP DATABASE marvin_staging;
-- DROP USER marvin_test;

CREATE USER marvin_test UNENCRYPTED PASSWORD 'marvin_test' ;
CREATE DATABASE marvin_test OWNER marvin_test;
CREATE DATABASE marvin_staging OWNER marvin_test;
GRANT ALL PRIVILEGES ON DATABASE marvin_test TO marvin_test ;
