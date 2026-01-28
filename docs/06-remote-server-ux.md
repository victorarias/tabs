# Remote Server UX Specification: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** SPEC

---

## Overview

This document specifies the user experience for the remote tabs server running at `https://tabs.company.com`.

**Design Principles:**
- **Transparency first** - All shared sessions visible to everyone by default
- **Discovery-focused** - Easy to find relevant sessions via search and tags
- **Read-only** - Sessions cannot be edited, only viewed
- **Team knowledge** - Emphasize learning from others' prompts
- **Minimal friction** - Fast loading, simple navigation

**Key Differences from Local UI:**
- **No "Share" button** - Sessions are already shared
- **Multi-user** - Shows who uploaded each session
- **Tag-based discovery** - Browse by team, repo, category
- **Search across all** - Search entire organization's sessions
- **API key management** - Create keys to push from local

---

## Authentication Flow

### IAP (Identity-Aware Proxy)

**User visits:** `https://tabs.company.com`

**Flow:**
```
User → IAP → Authenticate (SSO/Google/etc.) → tabs-server
```

**First-time users:**
1. User clicks "Create API Key" (protected by IAP)
2. IAP authenticates user
3. User creates API key with name (e.g., "My Laptop")
4. Key displayed once (must copy and save)
5. User configures key locally: `tabs-cli config set api-key ...`

**Returning users:**
- IAP handles authentication transparently
- User identity in header: `X-Forwarded-User: alice@company.com`
- Server logs user actions for audit

---

## Visual Design

### Same Design Language as Local UI

- Uses same typography, spacing, and grid system as local UI
- Consistent component library (Shadcn UI)
- Same animations and micro-interactions
- Users feel at home switching between local and remote

**Accent Color Variation:**
- **Local UI:** Amber (#d97706) - warm, personal
- **Remote UI:** Emerald (#059669) - shared, collaborative
- This subtle distinction helps users know which context they're in

### Key Visual Differences

**Navigation:**
- Added "API Keys" nav item (for key management)
- User avatar/name in top right (from IAP)

**Session Cards:**
- Added "Uploaded by" field (user who shared)
- Added "Uploaded at" timestamp
- Tags displayed prominently (not just in detail)

**Branding:**
- Company logo option (customizable)
- Optional: "Powered by tabs" footer

---

## Layout Structure

### App Shell

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]           [Search...🔍]   [Keys] [alice@...] [🌙]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Main Content                                               │
│                                                             │
│                                                             │
│                                                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Header:**
- Logo: "tabs" with music note
- Search bar (always visible)
- Navigation: API Keys
- User menu: Profile, Settings, Logout
- Theme toggle

---

## Page: Homepage (Session Timeline)

### URL
`https://tabs.company.com/`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]           [Search...🔍]   [Keys] [alice@...] [🌙]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Shared Sessions                                            │
│                                                             │
│  Filters: [All Tools ▾] [All Tags ▾] [Date ▾]      [142]   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Today                                               │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ 🤖 claude-code · 2:05 PM · 5m                       │   │
│  │ /home/user/projects/myapp                           │   │
│  │ Implement prime checking function                   │   │
│  │ 📤 alice@company.com · 2:10 PM                      │   │
│  │ 🏷️ team:platform · repo:myapp                       │   │
│  │ 12 messages · 8 tools                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🔮 cursor · 12:30 PM · 15m                          │   │
│  │ /home/bob/projects/webapp                           │   │
│  │ Fix authentication bug in login flow                │   │
│  │ 📤 bob@company.com · 12:45 PM                       │   │
│  │ 🏷️ team:frontend · repo:webapp · category:bugfix   │   │
│  │ 24 messages · 12 tools                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Yesterday                                           │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ ...                                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Load more...] (142 total sessions)                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Components

#### Session Card (Remote)

```
┌──────────────────────────────────────────────────────┐
│ 🤖 claude-code · 2:05 PM · 5m                        │
│ /home/user/projects/myapp                            │
│ Implement prime checking function                    │
│ 📤 alice@company.com · 2:10 PM                       │
│ 🏷️ team:platform · repo:myapp                        │
│ 12 messages · 8 tools                                │
└──────────────────────────────────────────────────────┘
```

**New Fields:**
- **Uploaded by:** User who shared (with avatar if available)
- **Uploaded at:** When it was shared (relative time)
- **Tags:** Displayed as pills, clickable to filter

**Interactions:**
- Click card → Navigate to session detail
- Click tag → Filter by that tag
- Click uploader → Filter by that user (future feature)
- Hover → Slight lift, no "Share" button (already shared)

#### Filters Bar

```
Filters: [All Tools ▾] [All Tags ▾] [Date ▾]    [Save Search]    [142 sessions]
```

**Tool Filter:**
- Dropdown: All, Claude Code, Cursor
- Shows count for each

**Tag Filter:**
- Dropdown with search: "Search tags..."
- Common tags listed first (team, repo, category)
- Multi-select (can filter by multiple tags: team:platform AND repo:myapp)
- Shows active tags as pills below filter bar

**Date Filter:**
- Same as local UI: Today, Yesterday, Last 7 days, etc.

**Save Search Button:**
- Saves current filters to localStorage
- Appears in "Saved Searches" dropdown
- Quick access to common queries

**Active Filters Display:**
```
Active filters: [team:platform ✕] [repo:myapp ✕] [Last 7 days ✕]
[Clear all]
```

#### Search Bar (Enhanced)

```
[🔍 Search across all shared sessions...                ]

Recent searches:
  - authentication bug
  - npm install error
  - team:platform refactor

Saved searches:
  ⭐ My team's sessions (team:platform)
  ⭐ Recent bugs (category:bugfix last 7 days)
```

**Features:**
- Search across all sessions (not just yours)
- Search includes: CWD, messages, tool names, file paths, tags, uploader names
- Recent searches (last 10, localStorage)
- Saved searches (pinned for quick access)
- Keyboard shortcut: `Cmd/Ctrl + K`

---

## Page: Session Detail

### URL
`https://tabs.company.com/sessions/:id`

### Layout

Same as local UI, with these additions:

**Header:**
```
Session: 550e8400-e29b-41d4-a716-446655440000
🤖 claude-code · 2026-01-28 2:05 PM · 5m
/home/user/projects/myapp

📤 Shared by alice@company.com on 2026-01-28 2:10 PM
🏷️ team:platform · repo:myapp

Stats: 12 messages · 8 tools · 3 files changed
```

**Actions:**
- **Copy session URL:** `https://tabs.company.com/sessions/123e4567...`
- **Copy session ID:** `550e8400-e29b-41d4-a716-446655440000`
- **View similar sessions:** Find sessions with same tags (future feature)

**No "Share" button** - Session is already shared

### Same Components as Local UI

- Message bubbles (user/assistant)
- Thinking blocks (collapsible)
- Tool cards (collapsible input/output)
- Syntax highlighting
- Copy buttons

---

## Page: API Keys

### URL
`https://tabs.company.com/keys`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]           [Search...🔍]   [Keys] [alice@...] [🌙]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  API Keys                                                   │
│                                                             │
│  Create keys to upload sessions from your local machine.   │
│                                                             │
│  [+ Create New Key]                                         │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ My Laptop                                           │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Key: tabs_abc1234••••••••••••••••••••               │   │
│  │ Created: 2026-01-28 10:00 AM                        │   │
│  │ Last used: 2 hours ago                              │   │
│  │ Usage: 42 sessions uploaded                         │   │
│  │                                                     │   │
│  │ [Revoke]                                            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Work Desktop                                        │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Key: tabs_def5678••••••••••••••••••••               │   │
│  │ Created: 2026-01-20 3:30 PM                         │   │
│  │ Last used: Never                                    │   │
│  │ Usage: 0 sessions uploaded                          │   │
│  │                                                     │   │
│  │ [Revoke]                                            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Old Laptop (Revoked)                                │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Key: tabs_ghi9012••••••••••••••••••••               │   │
│  │ Created: 2025-12-01                                 │   │
│  │ Revoked: 2026-01-15                                 │   │
│  │ Usage: 127 sessions uploaded                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Create New Key Modal

**Trigger:** Click "+ Create New Key"

**Modal:**
```
┌────────────────────────────────────────────────┐
│  Create API Key                       [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  Give this key a name to help you identify    │
│  which device it's for.                        │
│                                                │
│  Name:                                         │
│  [My Laptop                                  ] │
│                                                │
│  [Cancel]                       [Create Key]   │
│                                                │
└────────────────────────────────────────────────┘
```

**After Creation:**
```
┌────────────────────────────────────────────────┐
│  API Key Created ✓                    [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  ⚠️  Save this key now - it won't be shown    │
│     again!                                     │
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ tabs_abc123def456ghi789jkl012mno345pqr  │ │
│  │ [Copy to Clipboard]                      │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  Next steps:                                   │
│  1. Copy the key above                         │
│  2. On your local machine, run:                │
│     tabs-cli config set api-key <paste-key>    │
│  3. Start sharing sessions!                    │
│                                                │
│  [I've saved it, close this]                   │
│                                                │
└────────────────────────────────────────────────┘
```

**Key Security:**
- Key shown ONCE only
- After modal closes, key is permanently hidden
- User must save it (cannot retrieve later)

### Revoke Key

**Trigger:** Click "Revoke" on key card

**Confirmation Modal:**
```
┌────────────────────────────────────────────────┐
│  Revoke API Key?                      [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  Are you sure you want to revoke this key?    │
│                                                │
│  Key: tabs_abc1234•••••                        │
│  Name: My Laptop                               │
│                                                │
│  This action cannot be undone. You'll need to │
│  create a new key to upload sessions from     │
│  this device.                                  │
│                                                │
│  [Cancel]                    [Revoke Key]      │
│                                                │
└────────────────────────────────────────────────┘
```

**After Revocation:**
- Key card moves to bottom with "Revoked" badge
- Grayed out appearance
- No longer accepts uploads

---

## Page: Search Results

### URL
`https://tabs.company.com/search?q=authentication+bug&tag=team:platform`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]   [authentication bug        🔍]  [Keys] [alice@]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Search Results                                             │
│                                                             │
│  Found 12 sessions matching "authentication bug"            │
│  Filters: [team:platform ✕]                                │
│                                                             │
│  Sort by: [Relevance ▾]  [Date ▾]  [Duration ▾]            │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🤖 cursor · Yesterday 12:30 PM · 15m                │   │
│  │ /home/bob/projects/webapp                           │   │
│  │ Fix authentication bug in login flow                │   │
│  │ Match: "authentication bug" in user message         │   │
│  │ 📤 bob@company.com                                  │   │
│  │ 🏷️ team:platform · category:bugfix                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🤖 claude-code · 3 days ago · 8m                    │   │
│  │ /home/alice/projects/auth-service                   │   │
│  │ Debug JWT token validation                          │   │
│  │ Match: "authentication" in file path                │   │
│  │ 📤 alice@company.com                                │   │
│  │ 🏷️ team:platform · repo:auth-service                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Load more...] (12 results)                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Features

**Result Highlighting:**
- Search terms highlighted in session summary
- Show where match was found: "in user message", "in file path", "in tool output"

**Sorting:**
- **Relevance:** Default, based on search algorithm
- **Date:** Newest first or oldest first
- **Duration:** Longest or shortest sessions

**Saved Searches:**
- Button: "Save this search"
- Saves query + filters to localStorage
- Accessible from search bar dropdown

---

## Saved Searches (localStorage)

### Persistence

**Stored in browser localStorage:**
```json
{
  "savedSearches": [
    {
      "id": "search-1",
      "name": "My team's sessions",
      "query": "",
      "filters": {
        "tag": ["team:platform"]
      },
      "created_at": "2026-01-28T12:00:00Z"
    },
    {
      "id": "search-2",
      "name": "Recent bugs",
      "query": "",
      "filters": {
        "tag": ["category:bugfix"],
        "date": "last_7_days"
      },
      "created_at": "2026-01-28T13:00:00Z"
    }
  ]
}
```

### UI

**In Search Dropdown:**
```
┌────────────────────────────────────────┐
│ Saved Searches                         │
├────────────────────────────────────────┤
│ ⭐ My team's sessions                  │
│ ⭐ Recent bugs                          │
│ ⭐ Authentication fixes                 │
├────────────────────────────────────────┤
│ [Manage saved searches...]             │
└────────────────────────────────────────┘
```

**Manage Saved Searches Modal:**
```
┌────────────────────────────────────────────────┐
│  Saved Searches                       [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ ⭐ My team's sessions          [Edit] [✕]│ │
│  │ Query: (none)                            │ │
│  │ Filters: team:platform                   │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ ⭐ Recent bugs                 [Edit] [✕]│ │
│  │ Query: (none)                            │ │
│  │ Filters: category:bugfix, last 7 days   │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  [Close]                                       │
│                                                │
└────────────────────────────────────────────────┘
```

---

## Transparency & Discovery Features

### Browse by Tag

**Feature:** Tag cloud on homepage (sidebar or below filters)

```
Popular Tags:
┌────────────────────────────────────────┐
│ [team:platform (42)]                   │
│ [team:frontend (38)]                   │
│ [repo:myapp (67)]                      │
│ [category:bugfix (23)]                 │
│ [category:feature (31)]                │
│ [category:refactor (12)]               │
└────────────────────────────────────────┘
```

- Font size varies by count (tag cloud style)
- Click tag to filter by it
- Shows count in parentheses

### Recent Activity

**Widget on homepage (optional):**
```
┌────────────────────────────────────────┐
│ Recent Activity                        │
├────────────────────────────────────────┤
│ • alice@company.com shared 3 sessions  │
│   2 minutes ago                        │
│                                        │
│ • bob@company.com shared 1 session     │
│   15 minutes ago                       │
│                                        │
│ • carol@company.com shared 2 sessions  │
│   1 hour ago                           │
└────────────────────────────────────────┘
```

### Team Leaderboard (Optional, Fun)

**Widget showing most active sharers:**
```
┌────────────────────────────────────────┐
│ Top Sharers This Week 🏆               │
├────────────────────────────────────────┤
│ 1. alice@company.com    42 sessions    │
│ 2. bob@company.com      38 sessions    │
│ 3. carol@company.com    31 sessions    │
└────────────────────────────────────────┘
```

- Gamification to encourage sharing
- Optional, can be disabled

---

## Keyboard Shortcuts (Remote)

### Global

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + K` | Focus search |
| `Cmd/Ctrl + /` | Show keyboard shortcuts |
| `Esc` | Close modal / Clear search |

### Timeline

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate sessions |
| `Enter` | Open selected session |
| `/` | Focus search |
| `T` | Toggle tag filter |

### Session Detail

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + ←` | Back to timeline |
| `C` | Copy session URL |
| `T` | Toggle thinking blocks |

---

## Mobile Responsive

### Same as Local UI

- Mobile: Single column, collapsible filters
- Tablet: 2-column layout for timeline
- Desktop: Full layout with sidebar for filters

---

## Empty States

### No Sessions Shared Yet

```
┌────────────────────────────────────────┐
│                                        │
│         📋                             │
│         No sessions shared yet         │
│                                        │
│  Be the first to share a session!     │
│                                        │
│  1. Create an API key above            │
│  2. Configure it locally               │
│  3. Share a session from your local UI │
│                                        │
│  [Create API Key]                      │
│                                        │
└────────────────────────────────────────┘
```

### No Search Results

```
┌────────────────────────────────────────┐
│                                        │
│         🔍                             │
│         No results found               │
│                                        │
│  Try a different search term or        │
│  adjust your filters.                  │
│                                        │
│  [Clear filters]                       │
│                                        │
└────────────────────────────────────────┘
```

---

## Performance & Optimization

### Pagination

- Default: 20 sessions per page
- "Load more" button at bottom
- Virtual scrolling for 100+ sessions (same as local UI)

### Caching

- Session metadata cached in browser (sessionStorage)
- Full sessions loaded on demand (cache for 5 minutes)
- API responses cached with ETag support

### Database Query Optimization

- Indexes on: created_at, tool, tag_key + tag_value
- Full-text search on messages content (PostgreSQL `tsvector`)
- Pagination with LIMIT + OFFSET

---

## Analytics & Insights (Future)

### Session Statistics

**Page:** `https://tabs.company.com/stats`

**Widgets:**
- Total sessions shared
- Sessions per tool (pie chart)
- Sessions per team (bar chart)
- Most common tags (tag cloud)
- Most active contributors (leaderboard)
- Average session duration
- Most used tools (read, write, bash, etc.)

**Time-based filters:**
- Last 7 days
- Last 30 days
- All time

---

## Administration (Future)

### Admin Panel

**Page:** `https://tabs.company.com/admin` (restricted to admins)

**Features:**
- View all API keys (all users)
- Revoke any key
- Delete sessions (if needed for compliance)
- View audit log (who uploaded what, when)
- Configure retention policies

**Out of scope for v1** - Add in v2 if needed

---

## Security & Privacy

### Data Visibility

**Transparency-first approach:**
- All shared sessions visible to everyone with IAP access
- No private sessions
- No per-session permissions

**If privacy needed (future):**
- Add "team" field to sessions
- Filter sessions by user's teams
- Only show sessions from teams user belongs to

### Audit Logging

**Track:**
- Session uploads (who, when, from which API key)
- Session views (who viewed which session, when)
- API key creation/revocation
- Search queries (for analytics, not surveillance)

**Storage:**
- Separate audit log table in PostgreSQL
- Retained for 1 year (compliance)

---

## Error Handling

### Network Errors

**Session fails to load:**
- Show error banner: "Failed to load session. Try refreshing."
- Retry button

**Search fails:**
- Show error banner: "Search failed. Please try again."
- Fallback to cached results if available

### IAP Errors

**Authentication fails:**
- Redirect to IAP login page
- After login, redirect back to original page

**Authorization fails (no access):**
- Show 403 page: "Access Denied. Contact your admin."

---

## Conclusion

This remote server UX provides:
- ✅ **Transparency** - All sessions visible to everyone
- ✅ **Discovery** - Tag-based browsing, powerful search
- ✅ **API key management** - Easy to create and revoke keys
- ✅ **Read-only** - Sessions cannot be edited
- ✅ **Team knowledge** - Learn from others' prompts and approaches
- ✅ **Consistent design** - Matches local UI for familiarity

**Next Steps:**
1. Generate Daemon Implementation SPEC
2. Generate CLI Implementation SPEC
3. Generate Remote Server Implementation SPEC

---

**Document Status:** Ready for review
**Last Updated:** 2026-01-28
