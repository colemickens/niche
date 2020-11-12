#!/usr/bin/env bash

BUCKET="niche-test-$RANDOM"

aws s3 create

cat<<EOF >/tmp/s3.conf
{
    "": ""
}
EOF

aws s3 upload /tmp/s3.conf s3://niche-test-
