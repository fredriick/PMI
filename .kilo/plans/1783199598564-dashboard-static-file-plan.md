# Dashboard Layout Fix: Static File + Left Sidebar

## Context
`GET /dashboard` currently returns a minimal inline HTML page from `gateway/web/webui.go:DashboardHandler`. The rich dashboard UI in `web/dashboard/index.html` exists but is never served. Even after partial fixes, the horizontal navbar crowds and overlaps content.

## Goal
Serve the existing rich dashboard UI from `web/dashboard/index.html` and replace the horizontal navbar with a left sidebar for reliable spacing and easier navigation.

## Decisions
- Use static file serving (`c.File("web/dashboard/index.html")`) instead of inline HTML generation
- Replace top horizontal nav with fixed left sidebar (220px)
- Move controls (auto-refresh, notifications, theme, ws-status) to a topbar
- Keep all existing sections and API endpoints unchanged

## Tasks

### 1. gateway/web/webui.go
Replace `DashboardHandler` body with static file serve:
```go
func (w *WebUI) DashboardHandler(c *gin.Context) {
    c.File("web/dashboard/index.html")
}
```

### 2. gateway/service.go
Expose logo asset before proxy routes:
```go
g.router.StaticFile("/logo.svg", "web/logo.svg")
```

### 3. web/dashboard/index.html — restructure body
Replace `<header>` + `.nav` with:
- `<aside class="sidebar">` containing logo + vertical nav buttons
- `<header class="topbar">` containing auto-refresh, notifications, theme toggle, ws-status
- `<div class="main-content">` wrapping all sections

Keep all existing `showSection('...')` onclick handlers unchanged.

### 4. web/dashboard/index.html — add CSS
Add styles for:
- `.sidebar` — fixed left, 220px, full height, scrollable nav
- `.sidebar-nav` — vertical flex column, gap 4px
- `.sidebar-nav button` — full width, padded, active state
- `.topbar` — fixed top, left 220px, right 0
- `.topbar-inner` / `.topbar-controls` — flex row, right-aligned
- `.main-content` — margin-left 220px, padding-top 70px
- Body reset: remove `padding-top:100px`
- Mobile `@media(max-width:768px)`: sidebar becomes horizontal scroll row at top, topbar left=0, main-content margin-left=0

Remove or override old `.nav`, `header`, and `header .container` styles that belong to the old horizontal layout.

### 5. web/dashboard/index.html — preserve data loading
Ensure existing JS that fetches `/web/api/*` endpoints remains intact. No backend changes.

## Validation
1. `go run .`
2. Open `http://localhost:8000/dashboard`
3. Verify left sidebar with logo + stacked nav buttons
4. Verify topbar controls are visible and functional
5. Verify clicking nav switches sections without overlap
6. Verify content is offset right of sidebar with clear spacing
7. Resize to mobile and verify sidebar collapses to horizontal scroll

## Risk
Medium. Layout restructuring of static HTML/CSS only. No API or backend behavior changes. Revert path: restore previous `web/dashboard/index.html` from git if needed.

## Follow-up: Grid View Toggle + Larger Sidebar Logo

### Problem
User reports:
1. Pages are in list/table view only; needs grid/card view option
2. Sidebar logo is too small (currently 28px)

### Required Changes

#### 1. web/dashboard/index.html — larger sidebar logo
Update sidebar logo height from 28px to 40px:
`html
<img src="../logo.svg" alt="ProxyMesh" style="height:40px;width:auto">
`

#### 2. web/dashboard/index.html — Nodes grid view toggle
Add view toggle controls and grid rendering for the Nodes section.

**HTML changes in Nodes section:**
Add a view toggle button group next to the "Register Node" button:
`html
<div class="view-toggle">
  <button class="btn btn-ghost active" id="view-list" onclick="setNodeView('list')">List</button>
  <button class="btn btn-ghost" id="view-grid" onclick="setNodeView('grid')">Grid</button>
</div>
`

**CSS additions:**
`css
.view-toggle{
  display:flex;gap:4px;background:var(--bg-card);border:1px solid var(--border);
  border-radius:8px;padding:4px;
}
.view-toggle .btn{
  padding:4px 12px;font-size:0.78em;border-radius:6px;
}
.view-toggle .btn.active{
  background:var(--accent);color:#fff;border-color:var(--accent);
}
.nodes-grid{
  display:grid;
  grid-template-columns:repeat(auto-fill,minmax(280px,1fr));
  gap:16px;
}
.node-card{
  background:var(--bg-card);border:1px solid var(--border);border-radius:12px;
  padding:16px;position:relative;overflow:hidden;transition:border-color 0.3s,box-shadow 0.3s;
}
.node-card:hover{
  border-color:rgba(59,130,246,0.35);
  box-shadow:0 20px 60px rgba(0,0,0,0.4),0 0 40px var(--accent-dim);
}
.node-card-header{
  display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;
}
.node-card-title{
  font-family:'JetBrains Mono',monospace;font-weight:600;color:var(--accent-bright);font-size:0.9em;
}
.node-card-body{
  display:grid;grid-template-columns:1fr 1fr;gap:8px;font-size:0.85em;
}
.node-card-label{color:var(--text-dim);font-size:0.78em;text-transform:uppercase;letter-spacing:0.5px}
.node-card-value{color:var(--text);font-family:'JetBrains Mono',monospace}
.node-card-actions{
  margin-top:12px;padding-top:12px;border-top:1px solid var(--border);
  display:flex;gap:8px;justify-content:flex-end;
}
`

**JavaScript changes:**
Add state and toggle function:
`javascript
let nodeViewMode = 'list';

function setNodeView(mode) {
  nodeViewMode = mode;
  document.getElementById('view-list').classList.toggle('active', mode === 'list');
  document.getElementById('view-grid').classList.toggle('active', mode === 'grid');
  applyNodeFilters();
}

function renderFilteredNodes(nodes) {
  const container = document.getElementById('nodes-table').parentElement;
  if (nodeViewMode === 'grid') {
    container.innerHTML = '<div id="nodes-grid" class="nodes-grid"></div>';
    const grid = document.getElementById('nodes-grid');
    if (!nodes.length) {
      grid.innerHTML = '<div style="text-align:center;color:var(--text-dim);padding:20px;font-size:0.9em">No nodes match filters</div>';
      return;
    }
    grid.innerHTML = nodes.map(n => {
      const node = n.node || {};
      const lastSeen = node.last_seen ? new Date(node.last_seen).toLocaleString() : '--';
      const typeBadge = node.node_type === 'residential' ? '<span class="badge badge-yellow">Residential</span>' : '<span class="badge badge-green">Datacenter</span>';
      const dotColor = node.status === 'healthy' ? 'dot-green' : (node.status === 'warning' ? 'dot-yellow' : 'dot-red');
      return <div class="node-card" data-node-id="">
        <div class="node-card-header">
          <span class="node-card-title"></span>
          <span class="status-dot "></span>
        </div>
        <div class="node-card-body">
          <div><div class="node-card-label">Type</div><div class="node-card-value"></div></div>
          <div><div class="node-card-label">Country</div><div class="node-card-value"></div></div>
          <div><div class="node-card-label">City</div><div class="node-card-value"></div></div>
          <div><div class="node-card-label">ISP</div><div class="node-card-value"></div></div>
          <div><div class="node-card-label">Last Seen</div><div class="node-card-value"></div></div>
          <div><div class="node-card-label">Status</div><div class="node-card-value"></div></div>
        </div>
        <div class="node-card-actions">
          <button class="btn btn-danger" onclick="deregisterNode('')">Remove</button>
        </div>
      </div>;
    }).join('');
    updateNodeMap(nodes);
    return;
  }
  // Fallback to original table rendering
  container.innerHTML = '<table><thead><tr><th><input type="checkbox" id="select-all" onchange="toggleSelectAll(this)"></th><th class="sortable" onclick="sortNodes('status')">Status <span class="sort-indicator">\u2195</span></th><th class="sortable" onclick="sortNodes('id')">Node ID <span class="sort-indicator">\u2195</span></th><th class="sortable" onclick="sortNodes('type')">Type <span class="sort-indicator">\u2195</span></th><th class="sortable" onclick="sortNodes('country')">Country <span class="sort-indicator">\u2195</span></th><th>City</th><th>ISP</th><th class="sortable" onclick="sortNodes('lastSeen')">Last Seen <span class="sort-indicator">\u2195</span></th><th>Actions</th></tr></thead><tbody id="nodes-table"></tbody></table>';
  const tbody = document.getElementById('nodes-table');
  tbody.innerHTML = nodes.map(n => { ...same as original... }).join('');
  updateNodeMap(nodes);
}
`

Note: The grid view should preserve bulk actions (checkboxes) if possible, or show remove button per card. For simplicity, the card can show a Remove button per card and hide bulk selection in grid mode.

### Validation
1. Restart go run .
2. Open dashboard Nodes tab
3. Verify List view shows table as before
4. Click Grid toggle and verify cards appear in responsive grid
5. Verify sidebar logo is visibly larger
6. Verify toggle works back-and-forth

### Risk
Low. Only adds CSS/JS toggles and new card rendering path. Existing table path remains intact.

## Follow-up: Fix Dashboard Auth/WebSocket 401 Errors

### Problem
Console errors show 401 Unauthorized for:
1. GET /api/admin/theme - theme preference API
2. GET /favicon.ico - favicon
3. WebSocket connection to 'ws://localhost:8000/ws' failed - WS endpoint

These auth failures likely break JS execution (loadTheme throws, preventing other scripts from running), which explains why the grid/list toggle and tab navigation are not working despite the HTML/JS being present.

### Root Cause
The auth middleware in gateway/service.go has a public bypass list, but /api/admin/theme, /favicon.ico, and /ws are either missing or incorrectly handled.

### Required Changes

#### 1. gateway/service.go — auth bypass for dashboard public endpoints
Update uthMiddleware() (around line 181-192) to explicitly bypass:
- /api/admin/theme (GET and POST)
- /favicon.ico
- /ws

Ensure strings.HasPrefix(path, "/api/") covers /api/admin/theme, or add explicit matches.

#### 2. gateway/service.go — serve favicon statically
Add a static file route for favicon to prevent 401:
`go
g.router.StaticFile("/favicon.ico", "web/favicon.ico")
`
Or create a minimal web/favicon.ico if missing.

#### 3. gateway/service.go — WebSocket auth bypass
Ensure /ws route is registered before auth middleware or added to bypass list.

### Validation
1. go run .
2. Open http://localhost:8000/dashboard
3. Verify Console shows no 401 errors for /api/admin/theme, /favicon.ico, or /ws
4. Verify sidebar grid/list toggle buttons are visible and clickable
5. Verify tab navigation works
6. Verify Overview data loads

### Risk
Low. Only relaxes auth for non-sensitive public endpoints. Core proxy and admin APIs remain protected.

## Follow-up: Fix Dashboard 401 Errors for Theme, Favicon, and WebSocket

### Problem
Console shows 401 Unauthorized for:
- GET /api/admin/theme
- GET /favicon.ico
- WebSocket connection to 'ws://localhost:8000/ws'

These auth failures break JS execution and prevent dashboard features like the grid/list toggle from working.

### Root Cause
1. /api/admin/theme is bypassed by uthMiddleware() (matches strings.HasPrefix(path, "/api/")), but the handlers getThemeHandler/setThemeHandler internally check c.Get("user_id") and return 401 when it is missing.
2. /favicon.ico is not served statically and is not in the auth bypass list.
3. /ws is registered before uthMiddleware() is applied, but Gin still runs the middleware for it; since /ws is not in the bypass list, it returns 401.

### Required Changes

#### 1. gateway/service.go — make theme handlers public
Remove the user_id requirement from getThemeHandler and setThemeHandler so they work without authentication. Theme preference is not sensitive data.

In getThemeHandler (around line 736):
- Remove userID, exists := c.Get("user_id") and the 401 guard
- Keep the Redis theme lookup/fallback logic
- Optionally use a fixed/public userID like "dashboard" if you want to persist per-user theme in Redis

In setThemeHandler (around line 754):
- Remove userID, exists := c.Get("user_id") and the 401 guard
- Keep the theme validation and save logic

#### 2. gateway/service.go — auth bypass for favicon and WebSocket
Update uthMiddleware() bypass list to include:
`go
path == "/favicon.ico" || path == "/ws" ||
`

Place them alongside the existing public paths.

#### 3. gateway/service.go — serve favicon statically (optional but recommended)
Add before auth middleware:
`go
g.router.StaticFile("/favicon.ico", "web/favicon.ico")
`

If web/favicon.ico does not exist, create a minimal 1x1 transparent ICO or skip this step and rely on the auth bypass alone.

### Validation
1. go run .
2. Open http://localhost:8000/dashboard
3. Verify Console shows no 401 errors for /api/admin/theme, /favicon.ico, or /ws
4. Verify sidebar grid/list toggle buttons are visible and clickable
5. Verify tab navigation works
6. Verify Overview data loads

### Risk
Low. Only relaxes auth for non-sensitive public endpoints. Core proxy and admin APIs remain protected.

## Follow-up: Added Grid/List Toggle to All Dashboard Tabs

### Changes
- Added view toggle buttons (List/Grid) to every section: Nodes, Health, Sessions, Capacity, Users, Activity, Logs, API Keys
- Added CSS for .section-grid, .data-card, .data-card-header, .data-card-body, .data-card-label, .data-card-value, .data-card-actions
- Added grid rendering JS functions: enderHealthGrid, enderSessionsGrid, enderCapacityGrid, enderUsersGrid, enderActivityGrid, enderLogsGrid, enderAPIKeysGrid
- Added globalViewMode state and setViewMode(section, mode) function
- Updated showSection to preserve active toggle state when switching tabs
- Modified all load* functions to check globalViewMode and render either grid cards or table rows
- Fixed broken loadHealth function that was calling grid renderer without mode check

### Validation
- go vet ./gateway/... passes
- All 8 data tabs have List/Grid toggle buttons
- Grid mode renders responsive card layouts for each section type
- List mode falls back to original table rendering

### Risk
Low. Only adds UI toggle layer over existing data. No backend changes.
