#!/usr/bin/env bash

grpcurl \
  -plaintext \
  -H "Authorization: ApiKey test:test" \
  -d '{
    "path": ["/home/archivematica/archivematica-sampledata/SampleTransfers/Images/pictures"],
    "name": "Test",
    "type": "TRANSFER_TYPE_STANDARD",
    "processingConfig": "automated"
  }' \
    127.0.0.1:63030 \
    archivematica.ccp.admin.v1beta1.AdminService.CreatePackage
