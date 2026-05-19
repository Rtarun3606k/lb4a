#!/bin/bash

FILE=$1

if [ -z "$FILE" ]; then
  echo "Usage: ./analyze_logs.sh <logfile>"
  exit 1
fi

CLEAN=$(mktemp)
grep -E '^{.*}$' "$FILE" > "$CLEAN"

echo "================ FULL REQUEST TABLE ================"
jq -r '
select(.msg=="Access Log") |
[
  .time,
  .method,
  .path,
  .status,
  (if .status>=200 and .status<300 then "SUCCESS" else "FAIL" end),
  .duration
] | @tsv
' "$CLEAN" | column -t

echo ""
echo "================ HEALTH + REQUEST EVENTS ================"
jq -r '
if .msg=="Access Log" then
  [.time, .path, .status, "REQ", .duration]
elif .msg=="SERVER OFFLINE - Evicting from pool" then
  [.time, .url, "DOWN", "HEALTH_FAIL", .reason]
else empty end
| @tsv
' "$CLEAN" | column -t

echo ""
echo "================ STATUS COUNT ================"
jq -r '
select(.msg=="Access Log") |
(.status|tostring)
' "$CLEAN" | sort | uniq -c

echo ""
echo "================ ROUTE SUCCESS RATE ================"
jq -r '
select(.msg=="Access Log") |
"\(.path) \(.status)"
' "$CLEAN" | awk '
{
  total[$1]++
  if($2 ~ /^2/) success[$1]++
}
END {
  for (r in total)
    printf "%s -> %d/%d success (%.2f%%)\n",
    r, success[r], total[r], (success[r]/total[r])*100
}'

echo ""
echo "=================== TRUE SUCCESS P95 LATENCY ==================="
# Filters only 200/201 status codes and strips the "ms" cleanly
jq -r '
select(.msg=="Access Log" and .status >= 200 and .status < 300) |
.duration
' "$CLEAN" | sed 's/ms//' | sort -n | awk '
{arr[NR]=$1}
END {
  if (NR > 0) {
    p95=arr[int(NR*0.95) + 1]
    print "Success P95 latency:", p95 " ms"
  } else {
    print "Success P95 latency: N/A (No successful requests)"
  }
}'

rm "$CLEAN"
