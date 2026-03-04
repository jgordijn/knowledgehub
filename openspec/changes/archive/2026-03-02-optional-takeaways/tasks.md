## 1. Database Schema

- [x] 1.1 Add `takeaways` JSON field (nullable) to the `entries` collection in `cmd/knowledgehub/collections.go`
- [x] 1.2 Add `Takeaways` field to `testutil.CreateEntry` helper if needed

## 2. AI Summarizer

- [x] 2.1 Add `Takeaways []string` field to `SummaryResult` struct in `internal/ai/summarizer.go`
- [x] 2.2 Update `buildSummaryPrompt` to instruct the LLM to optionally include takeaways for longer/complex articles
- [x] 2.3 Update `parseSummaryResult` to handle optional `takeaways` field (null, missing, or present)
- [x] 2.4 Update `SummarizeAndScore` to store takeaways on the entry record
- [x] 2.5 Verify `ScoreOnly` path is unaffected (no takeaways for fragments)

## 3. Tests

- [x] 3.1 Add test case: summary result with takeaways parses correctly
- [x] 3.2 Add test case: summary result without takeaways parses correctly (backward compat)
- [x] 3.3 Add test case: `SummarizeAndScore` stores takeaways on entry when present
- [x] 3.4 Add test case: `SummarizeAndScore` stores null takeaways when absent
- [x] 3.5 Verify existing summarizer tests still pass

## 4. Frontend

- [x] 4.1 Update `EntryCard.svelte` to conditionally render takeaways as a compact bullet list below the summary
- [x] 4.2 Style takeaways list (small text, tight spacing, subtle appearance)
