# Dashboard UI Fix Plan

## Problem
Admin dashboard at `web/dashboard/index.html` has broken HTML structure causing:
- Nodes page UI corruption (unclosed divs, orphaned containers)
- Other pages not rendering (blank sections)
- Sidebar navigation present but sections fail to display

## Root Cause
Previous view-toggle/grid feature removal left:
1. Unclosed `<div>` tags in nodes section (depth imbalance)
2. Empty flex container divs from removed toggle buttons
3. Potential section wrapper divs not properly closed before next section

## Current State
- `web/dashboard/index.html` has 14 sections (overview, nodes, health, cooldowns, subnets, sessions, capacity, users, activity, benchmark, referrals, logs, config, apikeys)
- JavaScript syntax is valid (verified with `node --check`)
- CSS section visibility rules are correct: `.section { display: none }`, `.section.active { display: block }`
- `showSection()` function exists and targets sections by ID
- Nodes section has div depth imbalance when health section begins

## Fix Plan

### Step 1: Audit all sections for HTML balance
For each of the 14 sections, verify:
- Opening `<div id="..." class="section">` has matching `</div>`
- No orphaned child divs (empty containers, missing closes)
- Section boundaries are clean (next section starts after current section closes)

### Step 2: Fix nodes section structure
Specific issues in nodes section:
- Remove empty `<div style="display:flex;gap:8px;align-items:center">` left by view-toggle removal
- Ensure `section-header` div is properly closed
- Ensure main `section` div is properly closed before health section starts
- Verify table, forms, and cards are properly nested

### Step 3: Fix any other sections with similar issues
Apply same scrutiny to all sections, particularly:
- Sections that had view-toggle buttons removed
- Sections with complex nested layouts (users, apikeys, logs)

### Step 4: Clean up CSS and JS remnants
- Remove unused `.view-toggle` CSS rules
- Remove unused `globalViewMode` / `setViewMode` JS if present
- Ensure no references to removed grid functions

### Step 5: Validate
- Run `node --check` on extracted JavaScript
- Verify all 14 sections have balanced divs
- Build Go project (`go build ./...`)
- Serve dashboard and manually verify all sections render

## Files to Modify
- `web/dashboard/index.html` - fix HTML structure, remove remnants

## Validation
- `node --check` on extracted JS passes
- All 14 sections have balanced opening/closing divs
- `go build ./...` passes
- Dashboard loads at `http://localhost:8000/web/dashboard` with all sections visible via sidebar
