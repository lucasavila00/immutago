# Deployment checklist

Use a small release first. Confirm that the release has passed its tests and that
the team can restore the previous version.

## Before release

Read the change list. Check database compatibility and tell the on-call engineer
when the deployment will start. Keep the previous application image available.

## After release

Watch request errors and latency. Compare them with the values from before the
deployment. If an incident starts, use the incident recovery runbook. Do not
combine a rollback with a second untested change.
