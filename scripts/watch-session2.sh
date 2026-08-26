#!/bin/bash
# Watcher sesi 2 bursa: mulai 13:31 WIB, sampel 6x per menit.
# Bukti yang dikumpulkan: trade baru, pertumbuhan ob_book, key redis, sinyal.
Q="curl -s -u questdb:questdb"
echo "=== WATCHER SESI 2 — mulai $(date -u +%FT%TZ) ==="
while [ "$(date -u +%H%M)" -lt "0631" ]; do sleep 15; done
for i in 1 2 3 4 5 6; do
  ts=$(TZ=Asia/Jakarta date +%H:%M:%S)
  trades=$($Q --get --data-urlencode "query=select count() from running_trades where ts > dateadd('m',-1,now())" http://localhost:9000/exec \
    | python3 -c "import json,sys;d=json.load(sys.stdin);print(d['dataset'][0][0])" 2>/dev/null)
  ob=$($Q "http://localhost:9000/exec?query=select%20count(),count_distinct(symbol)%20from%20ob_book" \
    | python3 -c "import json,sys;d=json.load(sys.stdin);r=d['dataset'][0];print(str(r[0])+' baris / '+str(r[1])+' simbol')" 2>/dev/null)
  alerts=$(docker logs sbterm-ingest --since 2m 2>&1 | grep -c "bandarmology signal")
  keys=$(docker exec sbterm-redis redis-cli --scan --pattern 'ob:book:*' | wc -l)
  echo "[$ts] trades/menit=$trades | ob_book=$ob | redis_keys=$keys | sinyal(2m)=$alerts"
  sleep 55
done
echo "=== SELESAI $(date -u +%FT%TZ) ==="
