# Sample Input/Output

These samples demonstrate Zarvis's document parsing capabilities. Each input file is messy, unstructured data — the corresponding output shows the clean, structured JSON that Zarvis produces.

## Samples

### 1. Messy Employee CSV → Structured JSON

**Input:** [`input/messy_employees.csv`](input/messy_employees.csv)
- 10 rows with **5 different date formats** (MM/DD/YYYY, YYYY-MM-DD, "Month D YYYY", DD-Mon-YYYY, YYYY/MM/DD)
- Salary as `$85,000`, `92000`, `$78000`, `95000.00`, and `eighty thousand` (text!)
- Missing names, ages as `NOT_AVAILABLE`, incomplete email domains
- City abbreviations: NYC, SF mixed with full names

**Output:** [`output/messy_employees_structured.json`](output/messy_employees_structured.json)
- Clean array of employee objects with normalized fields
- Dates standardized to ISO 8601
- Currency symbols stripped, numbers parsed
- Per-row `issues` array flagging specific problems
- `dataQuality` summary: format counts, normalization map, missing value counts

---

### 2. Cloud Invoice → Structured JSON

**Input:** [`input/invoice_INV-2024-0847.txt`](input/invoice_INV-2024-0847.txt)
- Plain text with no schema — addresses, line items, tax calculations, payment details
- Mixed formatting: aligned columns, dollar amounts, quantity units

**Output:** [`output/invoice_structured.json`](output/invoice_structured.json)
- Nested structure: `invoice`, `from`, `to`, `lineItems[]`, `totals`, `payment`
- Every line item extracted with quantity, unit, unit price, and amount
- Tax rate calculated from subtotal and tax amount
- Payment routing details extracted

---

### 3. Server Crash Log → Structured JSON

**Input:** [`input/server_access.log`](input/server_access.log)
- 13 log entries with mixed formats: HTTP requests, system events, metrics
- Circuit breaker state transitions (OPEN → CLOSED)
- Embedded key-value pairs: `user_id=`, `amount=`, `err=`

**Output:** [`output/server_log_structured.json`](output/server_log_structured.json)
- Each entry parsed into typed fields: timestamp, level, method, path, status, duration
- Embedded values extracted: user IDs, amounts, error messages
- Summary computed: entries by level, unique endpoints, error rate, circuit breaker timeline

---

## Try it yourself

1. Start Zarvis: `docker compose up --build`
2. Register and upload any input file
3. Click "Parse into structured data" in the Explorer tab
4. Compare the output with the expected files above
5. Switch to Table/Schema/Summary tabs for different views
6. Export via the download button (JSON, CSV, TSV, Markdown)
