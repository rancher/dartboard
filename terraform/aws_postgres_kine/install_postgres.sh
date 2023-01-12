#!/usr/bin/env bash

set -xe

# Install the repository
mkdir -p /etc/yum.repos.d/
cat >/etc/yum.repos.d/pg.repo <<EOF
[pg15]
name=PostgreSQL 15 for RHEL/CentOS 7 - `uname -m`
baseurl=https://download.postgresql.org/pub/repos/yum/15/redhat/rhel-7-`uname -m`
enabled=1
gpgcheck=0
EOF

# Install PostgreSQL
yum install -y postgresql15 postgresql15-server

# Initialize the database and enable automatic start
ls /var/lib/pgsql/15/initdb.log || /usr/pgsql-15/bin/postgresql-15-setup initdb
systemctl enable postgresql-15
systemctl start postgresql-15

# Create kine user
su - postgres -c psql <<EOF
  CREATE USER kineuser WITH PASSWORD 'kinepassword';
  CREATE DATABASE kine LOCALE 'C' TEMPLATE 'template0';
  GRANT ALL ON DATABASE kine TO kineuser;
  ALTER DATABASE kine OWNER TO kineuser;
EOF

# Install kine
curl -L -o /usr/bin/kine https://github.com/k3s-io/kine/releases/download/v0.9.8/kine-`uname -m | sed 's/x86_64/amd64/'`
chmod +x /usr/bin/kine

cat >/etc/systemd/system/kine.service <<EOF
[Unit]
Description=kine

[Service]
ExecStart=/usr/bin/kine --endpoint postgres://kineuser:kinepassword@localhost:5432/kine?sslmode=disable

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload

# Start kine
systemctl enable kine
systemctl start kine
