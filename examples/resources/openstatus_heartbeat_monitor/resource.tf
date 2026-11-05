resource "openstatus_heartbeat_monitor" "nightly_backup" {
  name        = "Nightly backup"
  description = "Dumps the production database"
  period      = 86400 # the job runs once a day
  grace       = 1800  # and may take up to 30 minutes
  active      = true
}

# Pass the ping URL to the job, which requests it each time it runs:
#   /path/to/job.sh && curl -fsS -m 10 --retry 3 "$PING_URL" || curl -fsS -m 10 --retry 3 "$PING_URL/fail"
output "nightly_backup_ping_url" {
  value     = openstatus_heartbeat_monitor.nightly_backup.ping_url
  sensitive = true
}
