# Incident recovery

Use this runbook when requests fail or latency rises after a release.

## Restore service

Assign one incident lead. Record the affected service and the first failed request.
Check the deployment history before you change the system. If the latest release
caused the fault, roll back to the last working release.

## Check recovery

Check request success, latency, and the work queue. Ask the incident lead to confirm
recovery before you close the incident. Record the cause and the next action in
the incident report.
