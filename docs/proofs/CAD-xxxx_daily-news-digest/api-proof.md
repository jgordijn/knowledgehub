### GET /api/daily-news/settings
Request:
```http
GET /api/daily-news/settings HTTP/1.1
Authorization: Bearer <redacted>
```
Response:
```json
{
  "id": "3crsq62a80ltvau",
  "user": "cmeg2e4fzagp3dy",
  "enabled": true,
  "generation_time": "08:00",
  "timezone": "Europe/Amsterdam",
  "extra_instructions": ""
}
```

### PUT /api/daily-news/settings
Request:
```http
PUT /api/daily-news/settings HTTP/1.1
Authorization: Bearer <redacted>
Content-Type: application/json

{"enabled":true,"generation_time":"09:15","timezone":"Europe/Amsterdam","extra_instructions":"Prefer concise summaries with architecture and product impact."}
```
Response:
```json
{
  "id": "3crsq62a80ltvau",
  "user": "cmeg2e4fzagp3dy",
  "enabled": true,
  "generation_time": "09:15",
  "timezone": "Europe/Amsterdam",
  "extra_instructions": "Prefer concise summaries with architecture and product impact."
}
```

### GET /api/daily-news/digests/{digestId}
Request:
```http
GET /api/daily-news/digests/krda2x0mvdczsfd HTTP/1.1
Authorization: Bearer <redacted>
```
Response:
```json
{
  "id": "krda2x0mvdczsfd",
  "user": "cmeg2e4fzagp3dy",
  "status": "success",
  "trigger": "manual",
  "local_date": "2026-05-09",
  "title": "Daily News — proof digest",
  "body_markdown": "# Daily News — proof digest\n\n## Top stories\n\n- AI governance checklist is ready [[kh-entry:ceim5iri2be30us]]\n- Platform teams can prioritise with signal loops [[kh-entry:n8be2z8ampza0em]]\n\n<script>alert(1)</script>\n\n[blocked](javascript:alert(1))",
  "referenced_entry_ids": ["ceim5iri2be30us", "n8be2z8ampza0em"],
  "candidate_count": 2,
  "included_count": 2,
  "used_subset": false,
  "generated_at": "2026-05-09 08:00:00.000Z",
  "last_success_at": "2026-05-09 08:00:00.000Z",
  "has_successful_snapshot": true,
  "attempt_finished_at": "2026-05-09 08:00:00.000Z",
  "period_start": "2026-05-08 22:00:00.000Z",
  "period_end": "2026-05-09 22:00:00.000Z"
}
```

### GET /api/daily-news/digests/{digestId}/entries/{entryId}
Request:
```http
GET /api/daily-news/digests/krda2x0mvdczsfd/entries/ceim5iri2be30us HTTP/1.1
Authorization: Bearer <redacted>
```
Response:
```json
{
  "available": true,
  "entry": {
    "id": "ceim5iri2be30us",
    "title": "EU AI Act implementation guide",
    "url": "https://example.com/ai-act",
    "summary": "A concise explanation of implementation milestones, governance steps, and risk controls for AI systems.",
    "takeaways": ["Risk classification needs ownership", "Documentation and monitoring are required"],
    "effective_stars": 5,
    "source_name": "Proof RSS",
    "published_at": "2026-05-09 06:30:00.000Z",
    "discovered_at": "2026-05-09 06:40:00.000Z"
  }
}
```

### POST /api/daily-news/generate
Request:
```http
POST /api/daily-news/generate HTTP/1.1
Authorization: Bearer <redacted>
```
Response:
```http
HTTP/1.1 202 Accepted
Content-Type: application/json
```
```json
{
  "id": "acph1u4zot8lo4o",
  "user": "cmeg2e4fzagp3dy",
  "status": "pending",
  "trigger": "manual",
  "local_date": "2026-05-09",
  "candidate_count": 0,
  "included_count": 0,
  "used_subset": false,
  "has_successful_snapshot": false,
  "queued_at": "2026-05-09 06:18:35.000Z",
  "period_start": "2026-05-09 22:00:00.000Z",
  "period_end": "2026-05-09 06:18:35.000Z"
}
```
