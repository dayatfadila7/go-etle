#!/usr/bin/env bash
# etle-monitor.sh - monitoring kinerja sistem & database go-etle.
# Dipanggil tiap 5 menit via cron (/etc/cron.d/go-etle).
# Menulis dua file log TSV:
#   /var/log/etle-system-perf.log : CPU, RAM, disk, antrean kirim, ringkasan data
#   /var/log/etle-db-perf.log      : statistik kinerja PostgreSQL
set -u

source /opt/go-etle/.env 2>/dev/null || true
export PGPASSWORD="${DB_PASSWORD:-}"
PGHOST="${DB_HOST:-localhost}"; PGUSER="${DB_USERNAME:-}"; PGDBN="${DB_NAME:-}"

TS=$(date '+%Y-%m-%d %H:%M:%S')
SYSTEM_FILE=/var/log/etle-system-perf.log
DB_FILE=/var/log/etle-db-perf.log
HDR_SYS='time	load1	load5	cpu_pct_goetle	rss_mb_goetle	mem_used_pct	disk_used_pct	queue_pending	queue_failed	sent_total	total_rows'
HDR_DB='time	conns	xact_rollback	tup_inserted	tup_updated	tup_deleted	deadlocks	cache_hit_pct	db_size_mb'

if [ ! -s "$SYSTEM_FILE" ]; then printf '%s\n' "$HDR_SYS" > "$SYSTEM_FILE"; fi
if [ ! -s "$DB_FILE" ]; then printf '%s\n' "$HDR_DB" > "$DB_FILE"; fi

LOAD=$(awk '{printf "%s\t%s", $1, $2}' /proc/loadavg 2>/dev/null || printf '0\t0')
GOETLE=$(ps -C go-etle -o pcpu=,rss= 2>/dev/null | awk '{c+=$1; r+=$2} END {printf "%s\t%d", c, r/1024}')
MEM=$(free -m 2>/dev/null | awk '/^Mem:/ {printf "%d", $3*100/$2}')
DISK=$(df -P /home/data_pelanggaran / 2>/dev/null | awk 'NR>1 {u+=$3; t+=$2} END {printf "%.1f", u*100/t}')

QUEUE="0	0	0	0"
DBSTAT="0	0	0	0	0	0	0	0"
if command -v psql >/dev/null 2>&1; then
  QUEUE=$(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDBN" -At -F $'\t' -c "SELECT count(*) FILTER (WHERE status IN ('pending','processing')), count(*) FILTER (WHERE status='failed'), count(*) FILTER (WHERE status='sent'), count(*) FROM violations;" 2>/dev/null | head -1)
  [ -n "$QUEUE" ] || QUEUE="0	0	0	0"
  DBSTAT=$(psql -h "$PGHOST" -U "$PGUSER" -d "$PGDBN" -At -F $'\t' -c "SELECT (SELECT count(*) FROM pg_stat_activity), d.xact_rollback, d.tup_inserted, d.tup_updated, d.tup_deleted, d.deadlocks, CASE WHEN d.blks_hit+d.blks_read>0 THEN round(100*d.blks_hit/(d.blks_hit+d.blks_read),1) ELSE 0 END, round(pg_database_size(current_database())/1048576.0,1) FROM pg_stat_database d WHERE d.datname=current_database();" 2>/dev/null | head -1)
  [ -n "$DBSTAT" ] || DBSTAT="0	0	0	0	0	0	0	0"
fi

printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$TS" "$LOAD" "$GOETLE" "$MEM" "$DISK" "$QUEUE" >> "$SYSTEM_FILE"
printf '%s\t%s\n' "$TS" "$DBSTAT" >> "$DB_FILE"