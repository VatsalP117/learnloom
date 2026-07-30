# Restore evidence

Quarterly restore evidence belongs in this directory and must be reviewed before
a production release. Run `scripts/restore-drill.sh` against an empty, isolated
database whose name contains `restore` or `drill`. Supply independently
downloaded source and restored artifact files so the harness verifies both
Postgres and object storage.

Evidence files must contain no database URL, credentials, learner content,
email address, prompt, or source body. A passing local drill does not prove
managed-provider PITR, IAM, replication, or backup retention; attach those
provider records separately in the release system.
