## Parked decison 1

**Plan:** when a late response chunk arrives for a stream whose capture has already been flushed (Get returns ok = false), log it for visibility rather than silently dropping it. No mechanism yet ties it back to the row, id is auto-increment and stream_id isn't a stored column. 

**Revisit later:** either add stream_id as a column to enable late-update, or accept the loss as inherent to the timeout boundary.


## Parked Decision 2
M3 Body parsing assumes one request per connection. capture.Request/capture.Response are parsed assuming exactly one HTTP message exists in the buffer. A keep-alive visitor sending multiple requests on one connection breaks this as the current code slices everything after the header block to end-of-buffer as the body, so a second request's bytes get silently absorbed into the first request's body field.

**Considered and rejected as a fix:** using Content-Length to correctly bound just the first request's body. This only masks the symptom, request 1 would parse cleanly, but request 2 (and any further pipelined requests) would still be entirely discarded, with no visible sign anything was dropped. A real fix needs parseRequest to loop and extract every message in the buffer, plus Flush/schema to support multiple request/response rows per capture. That will be done later

**v1 assumption:** one request per stream/connection

## Parked Decision 3
M3 -  No retry or preservation on failed Flush. If st.db.Exec's INSERT fails inside Flush, the capture is still deleted from active and the error is only logged. The in-memory data is lost with no retry. 

**Alternative consideration:** skip the delete and keep the entry in active to preserve the data. This is currently rejected for v1 because nothing currently revisits or retries a failed flush, so a kept entry would just leak forever instead of being cleaned up. _busy_timeout=5000 should make write-lock failures rare in practice. **Revisit retry/dead-letter mechanism later**