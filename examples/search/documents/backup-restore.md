# Backup restore checks

A successful backup job does not show that a restore will work. Run a restore
check in a separate test environment each week.

## Restore a copy

Choose the latest complete backup. Restore the database and compare the record
count with the recorded source count. Run a read request against the restored
application.

## Record the result

Record the backup time, restore time, and test result. Keep these values with the
recovery plan so the incident lead can estimate the time needed to restore data.
