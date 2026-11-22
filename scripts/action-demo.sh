#!/usr/bin/env sh
# Copyright 2026 Zyvor AI Labs · https://zyvor.dev
# SPDX-License-Identifier: LicenseRef-Zyvor-Production-2.0

set -eu
curl -sS -X POST http://127.0.0.1:8080/v1/actions \
  -H 'content-type: application/json' \
  -d '{"id":"demo-rpm-1","device_id":"pump-17","target":"rpm_setpoint","value":1450}'
printf '\n'
