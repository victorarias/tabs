# Local UI Flows Specification: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** SPEC

> **⚠️ IMPORTANT: Design System Update**
>
> The visual design language (typography, colors, animations) in this document has been superseded by **docs/07-frontend-design-brief.md**.
>
> **Use this document (05) for:** Page layouts, component structure, user flows
>
> **Use doc 07 for:** Typography (DM Serif Display + IBM Plex Sans), colors (vinyl & mahogany palette), animations (musical micro-interactions), spacing system
>
> When there's a conflict, doc 07 wins.

---

## Overview

This document specifies the user experience for the local tabs web UI running at `http://localhost:3787`.

**Design Principles:**
- **Fast and lightweight** - Direct filesystem reads, no heavy processing
- **Clean and tasteful** - Minimal, focused interface
- **Whimsical touches** - Small delightful animations and details
- **Keyboard-friendly** - Support keyboard navigation
- **Search-first** - Powerful search with saved filters

**Tech Stack:**
- **Framework:** TanStack Start (SSR + React)
- **Styling:** Tailwind CSS
- **Components:** Shadcn UI (headless, customizable)
- **Icons:** Lucide React
- **Animations:** Framer Motion (subtle, tasteful)
- **Code Highlighting:** Shiki or Prism

---

## Visual Design Language

### Color Palette

**Light Mode (Default):**
```css
--background: #ffffff
--foreground: #0a0a0a
--muted: #f5f5f5
--muted-foreground: #737373
--accent: #f5f5f5
--accent-foreground: #0a0a0a
--border: #e5e5e5

--primary: #2563eb     /* Blue for actions */
--primary-hover: #1d4ed8
--success: #16a34a     /* Green for success states */
--error: #dc2626       /* Red for errors */
--warning: #ea580c     /* Orange for warnings */
```

**Dark Mode:**
```css
--background: #0a0a0a
--foreground: #fafafa
--muted: #171717
--muted-foreground: #a3a3a3
--accent: #171717
--accent-foreground: #fafafa
--border: #262626

--primary: #3b82f6
--primary-hover: #2563eb
--success: #22c55e
--error: #ef4444
--warning: #f97316
```

### Typography

```css
--font-sans: 'Inter', system-ui, sans-serif
--font-mono: 'JetBrains Mono', 'Fira Code', monospace

/* Sizes */
--text-xs: 0.75rem     /* 12px */
--text-sm: 0.875rem    /* 14px */
--text-base: 1rem      /* 16px */
--text-lg: 1.125rem    /* 18px */
--text-xl: 1.25rem     /* 20px */
--text-2xl: 1.5rem     /* 24px */
--text-3xl: 1.875rem   /* 30px */
```

### Spacing

```css
--space-1: 0.25rem     /* 4px */
--space-2: 0.5rem      /* 8px */
--space-3: 0.75rem     /* 12px */
--space-4: 1rem        /* 16px */
--space-6: 1.5rem      /* 24px */
--space-8: 2rem        /* 32px */
--space-12: 3rem       /* 48px */
```

### Animations

**Principles:**
- Subtle, not distracting
- Fast (200-300ms)
- Purposeful (guide attention, indicate state changes)

**Common Animations:**
```css
/* Fade in */
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* Slide up */
@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Scale in */
@keyframes scaleIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
```

---

## Layout Structure

### App Shell

```
┌─────────────────────────────────────────────────────────────┐
│  Header (sticky)                                            │
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
- Logo: "tabs" with music note icon (🎵 or guitar pick)
- Navigation: Home, Settings
- Search bar (always visible)
- Theme toggle (light/dark)

**Main Content:**
- Single-column layout
- Max width: 1200px
- Centered on page
- Padding: 32px horizontal, 24px vertical

---

## Page: Homepage (Timeline View)

### URL
`http://localhost:3787/`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]           [Search...🔍]      [Settings] [🌙]    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Filters: [All ▾] [Date ▾] [Folder ▾]              [42]    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Today                                               │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ 🤖 12:05 PM · claude-code · 5m                      │   │
│  │ /home/user/projects/myapp                           │   │
│  │ Implement prime checking function                   │   │
│  │ 12 messages · 8 tools · 3 files changed             │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🤖 10:30 AM · cursor · 15m                          │   │
│  │ /home/user/projects/webapp                          │   │
│  │ Fix authentication bug in login flow                │   │
│  │ 24 messages · 12 tools · 5 files changed            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Yesterday                                           │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ 🤖 4:22 PM · claude-code · 12m                      │   │
│  │ ...                                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Components

#### Session Card

**Structure:**
```
┌──────────────────────────────────────────────────────┐
│ 🤖 12:05 PM · claude-code · 5m          [Share →]   │
│ /home/user/projects/myapp                            │
│ Implement prime checking function                    │
│ 12 messages · 8 tools · 3 files changed              │
└──────────────────────────────────────────────────────┘
```

**Details:**
- **Icon:** 🤖 for claude-code, 🔮 for cursor
- **Timestamp:** Relative time (5m ago, 2h ago, Yesterday)
- **Tool badge:** Pill with tool name, colored (blue for claude-code, purple for cursor)
- **Duration:** Human-readable (5m, 1h 15m)
- **CWD:** Truncated path (show last 2-3 segments), tooltip shows full path
- **Summary:** First user message or generated summary (1 line, truncated)
- **Stats:** Message count, tool count, files changed
- **Share button:** Appears on hover

**Hover State:**
- Slight scale up (1.02)
- Border color change
- Share button fades in
- Shadow increases

**Click:**
- Navigate to session detail page
- Smooth transition (fade out list, fade in detail)

#### Filters Bar

```
Filters: [All ▾] [Date ▾] [Folder ▾]              [42 sessions]
```

**Tool Filter:**
- Dropdown: All, Claude Code, Cursor
- Shows count next to each option

**Date Filter:**
- Dropdown: All, Today, Yesterday, Last 7 days, Last 30 days, Custom range
- Custom range opens date picker modal

**Folder Filter:**
- Dropdown: All, Recent folders (top 10)
- Shows folder path, grouped by parent

**Session Count:**
- Shows total count after filters applied
- Updates live as filters change

#### Search Bar

```
[🔍 Search sessions, messages, files...              ]
```

**Features:**
- Autofocus on page load (cmd+k also focuses)
- Placeholder text rotates:
  - "Search sessions, messages, files..."
  - "Try: npm install"
  - "Try: authentication bug"
  - "Try: /home/user/projects/myapp"
- Debounced search (300ms)
- Clear button (X) appears when text entered

**Search Behavior:**
- Searches across:
  - Session CWD
  - User messages
  - Assistant messages
  - Tool names
  - File paths in tools
- Highlights matches in results
- Shows "No results" state with suggestions

**Saved Searches (Future):**
- Saved to localStorage
- Quick access dropdown below search bar

---

## Page: Session Detail

### URL
`http://localhost:3787/sessions/:id`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [← Back]                            [Share] [🌙]          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Session: 550e8400-e29b-41d4-a716-446655440000              │
│  🤖 claude-code · 2026-01-28 12:00 PM · 5m                  │
│  /home/user/projects/myapp                                  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 👤 User · 12:00 PM                                  │   │
│  │ Please implement a function to check if a number    │   │
│  │ is prime                                            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🤖 Assistant · 12:00 PM                             │   │
│  │ [💭 Thinking...]                                     │   │
│  │ I'll create a function that checks if a number is   │   │
│  │ prime using trial division up to the square root.   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🔧 write · 12:00 PM                     [View File] │   │
│  │ ├─ Input                                            │   │
│  │ │  file_path: /home/user/projects/myapp/src/prime.ts│  │
│  │ │  content: export function isPrime(n: number) ...  │   │
│  │ ├─ Output                                           │   │
│  │ │  File written successfully                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ...                                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Components

#### Session Header

```
Session: 550e8400-e29b-41d4-a716-446655440000
🤖 claude-code · 2026-01-28 12:00 PM · 5m
/home/user/projects/myapp

Stats: 12 messages · 8 tools · 3 files changed
```

**Elements:**
- Session ID (small, monospace, copyable on click)
- Tool badge
- Start timestamp (absolute)
- Duration
- CWD (copyable, with copy icon on hover)
- Stats summary

**Actions:**
- Share button (top right)
- Copy session ID button
- Copy session URL button

#### Message Bubble (User)

```
┌──────────────────────────────────────────────────────┐
│ 👤 User · 12:00 PM                                   │
│ Please implement a function to check if a number     │
│ is prime                                             │
└──────────────────────────────────────────────────────┘
```

**Style:**
- Background: light gray (light mode), dark gray (dark mode)
- Border: subtle
- Padding: 16px
- Margin: 8px 0
- Font: sans-serif, 16px

#### Message Bubble (Assistant)

```
┌──────────────────────────────────────────────────────┐
│ 🤖 Assistant · 12:00 PM                              │
│ [💭 Thinking] (collapsible)                          │
│ I'll create a function that checks if a number is    │
│ prime using trial division up to the square root.    │
└──────────────────────────────────────────────────────┘
```

**Thinking Block:**
- Collapsed by default
- Click to expand/collapse
- Icon changes: 💭 (collapsed) → 🧠 (expanded)
- Slightly different background color (more muted)
- Italic text

**Content:**
- Markdown rendering (bold, italic, code)
- Syntax-highlighted code blocks
- Links clickable

#### Tool Use Card

```
┌──────────────────────────────────────────────────────┐
│ 🔧 write · 12:00 PM                     [View File]  │
│ ├─ Input  ▾                                          │
│ │  file_path: /home/user/projects/myapp/src/prime.ts│
│ │  content: export function isPrime(n: number) ...   │
│ ├─ Output ▾                                          │
│ │  File written successfully                         │
└──────────────────────────────────────────────────────┘
```

**Tool Icon:**
- 🔧 write
- 📖 read
- 💻 bash
- 🔍 grep/glob
- 🌐 webfetch
- 🎯 generic (for others)

**Collapsible Sections:**
- Input and Output are collapsible independently
- Collapsed by default if content is large (>500 chars)
- Expanded by default if small

**File Path Links:**
- File paths are clickable
- Copy to clipboard on click
- Show "Copied!" tooltip

**Code Blocks:**
- Syntax highlighted based on file extension or language
- Copy button in top-right corner
- Line numbers for large blocks (>10 lines)

**Error State:**
- If `is_error: true`, output has red border and background
- Error icon: ❌

---

### Interactions

#### Share Modal

**Trigger:** Click "Share" button

**Modal:**
```
┌────────────────────────────────────────────────┐
│  Share Session                        [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  Session Summary:                              │
│  🤖 claude-code · 2026-01-28 12:00 PM          │
│  /home/user/projects/myapp                     │
│  Duration: 5m · 12 messages · 8 tools          │
│                                                │
│  ─────────────────────────────────────────────  │
│                                                │
│  Tags (optional):                              │
│  [+ Add tag]                                   │
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ team: platform                    [✕]    │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ repo: myapp                       [✕]    │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  ─────────────────────────────────────────────  │
│                                                │
│  [Cancel]                         [Share →]    │
│                                                │
└────────────────────────────────────────────────┘
```

**Tag Input:**
- Format: `key:value`
- Autocomplete from existing tags
- Common tags suggested: team, repo, category, user
- Multiple tags allowed

**Share Flow:**
1. Click "Share"
2. Modal opens with session summary
3. User adds tags (optional)
4. Click "Share →"
5. Loading spinner appears
6. On success:
   - Modal shows success state
   - URL is displayed
   - "Copy URL" button
   - "View on server" button
7. Auto-close after 3 seconds, or user clicks X

**Success State:**
```
┌────────────────────────────────────────────────┐
│  Session Shared! ✓                    [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  Your session has been uploaded to:            │
│                                                │
│  https://tabs.company.com/sessions/123e4567... │
│  [Copy URL]                                    │
│                                                │
│  [View on server →]                            │
│                                                │
└────────────────────────────────────────────────┘
```

**Error State:**
```
┌────────────────────────────────────────────────┐
│  Share Failed ✗                       [✕]      │
├────────────────────────────────────────────────┤
│                                                │
│  Could not upload session:                     │
│  API key not configured                        │
│                                                │
│  [Go to Settings]            [Try Again]       │
│                                                │
└────────────────────────────────────────────────┘
```

---

## Page: Settings

### URL
`http://localhost:3787/settings`

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  [🎵 tabs]                                   [🌙]           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Settings                                                   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Remote Server                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Server URL                                          │   │
│  │ [https://tabs.company.com                         ] │   │
│  │                                                     │   │
│  │ API Key                                             │   │
│  │ [tabs_abc1234••••••••••••••••••••]  [Show] [Test]  │   │
│  │                                                     │   │
│  │ Status: ✓ Connected                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Local Storage                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Location: ~/.tabs/sessions/                         │   │
│  │ Size: 127 MB (42 sessions)                          │   │
│  │                                                     │   │
│  │ [Open in Finder]  [Clean up old sessions...]       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Daemon Status                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Status: ✓ Running (PID 12345)                       │   │
│  │ Uptime: 2h 15m                                      │   │
│  │ Sessions captured: 42                               │   │
│  │ Events processed: 1,337                             │   │
│  │                                                     │   │
│  │ [View Logs]  [Restart Daemon]                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Appearance                                          │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ Theme: ◉ Light  ◯ Dark  ◯ System                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Save Changes]                                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Sections

#### Remote Server

**Fields:**
- **Server URL:** Text input, validates HTTPS URL
- **API Key:** Password input (masked), "Show" button to reveal, "Test" button to verify

**Test Connection:**
- Sends request to server: `GET /api/sessions?limit=1`
- Shows loading spinner
- On success: Green checkmark, "✓ Connected"
- On error: Red X, "✗ Failed: [error message]"

**Get API Key Button:**
- Link: "Don't have an API key? Get one →"
- Opens remote server in browser: `https://tabs.company.com/keys`

#### Local Storage

**Info:**
- Location (not editable)
- Size in MB
- Session count

**Actions:**
- **Open in Finder:** Opens `~/.tabs/sessions/` in file manager
- **Clean up:** Opens modal to delete sessions older than X days

#### Daemon Status

**Info:**
- Running status (green dot + "Running" or red dot + "Stopped")
- PID
- Uptime
- Statistics (sessions captured, events processed)

**Actions:**
- **View Logs:** Opens modal with tail of `~/.tabs/daemon.log`
- **Restart Daemon:** Stops and starts daemon

#### Appearance

**Theme Selector:**
- Radio buttons: Light, Dark, System
- Persisted to localStorage
- Immediate preview

---

## Keyboard Shortcuts

### Global

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + K` | Focus search |
| `Cmd/Ctrl + /` | Show keyboard shortcuts help |
| `Esc` | Close modal / Clear search |
| `Cmd/Ctrl + ,` | Open settings |

### Timeline View

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate sessions |
| `Enter` | Open selected session |
| `/` | Focus search |

### Session Detail

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + ←` | Back to timeline |
| `S` | Share session |
| `T` | Toggle thinking blocks |
| `C` | Copy session ID |

---

## Responsive Design

### Breakpoints

```css
/* Mobile */
@media (max-width: 640px) { ... }

/* Tablet */
@media (min-width: 641px) and (max-width: 1024px) { ... }

/* Desktop */
@media (min-width: 1025px) { ... }
```

### Mobile Adaptations

**Timeline:**
- Single column
- Smaller session cards
- Filters collapse into hamburger menu
- Search bar full width

**Session Detail:**
- Header becomes sticky
- Tool cards stack vertically
- Code blocks horizontal scroll if needed

---

## Loading States

### Page Load

**Timeline:**
- Show skeleton cards (3-5 gray boxes)
- Fade in real content when loaded

**Session Detail:**
- Show loading spinner in center
- Fade in content when loaded

### Async Actions

**Share:**
- Button shows spinner, text changes to "Sharing..."
- Disable button during request

**Test Connection:**
- Show spinner next to "Test" button
- Update status icon when complete

---

## Empty States

### No Sessions

```
┌────────────────────────────────────────┐
│                                        │
│         📋                             │
│         No sessions yet                │
│                                        │
│  Start using Claude Code or Cursor    │
│  and your sessions will appear here.  │
│                                        │
│  [Learn more]                          │
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

## Whimsical Touches

### Music Theme

**Logo:**
- Guitar pick icon or music note (🎵)
- Subtle rotation on hover
- Click to play a note sound (optional, toggleable)

**Loading States:**
- Loading spinner could be rotating guitar pick
- Or music note bouncing

**Success Animations:**
- Share success: confetti (brief, 1 second)
- Copy success: checkmark with bounce

### Micro-interactions

**Session Cards:**
- Slight lift on hover (box shadow grows)
- Smooth transition (200ms ease-out)

**Buttons:**
- Scale down on click (0.95)
- Ripple effect on primary buttons

**Thinking Blocks:**
- Expand/collapse with smooth height transition
- Icon rotates 180° when expanding

**Code Blocks:**
- Copy button fades in on hover
- Success checkmark replaces button briefly when copied

---

## Accessibility

### ARIA Labels

- All interactive elements have `aria-label`
- Modals have `role="dialog"`
- Search has `role="search"`

### Keyboard Navigation

- All actions accessible via keyboard
- Focus indicators visible (blue outline)
- Tab order logical (top to bottom, left to right)

### Screen Readers

- Session cards announced with full context
- Tool names and results read aloud
- Loading states announced

### Color Contrast

- All text meets WCAG AA standards (4.5:1 minimum)
- Interactive elements have 3:1 contrast with background

---

## Performance

### Optimization Strategies

**Timeline:**
- Virtual scrolling for 100+ sessions
- Lazy load session details on demand
- Debounced search (300ms)

**Session Detail:**
- Lazy load code highlighting (only visible blocks)
- Collapse large tool outputs by default
- Image lazy loading (if screenshots added in future)

**Caching:**
- Session metadata cached in memory
- Full sessions loaded on demand
- LocalStorage for user preferences

---

## Error Handling

### Network Errors

**Push fails:**
- Show error modal with retry option
- Suggest checking API key in settings

**Daemon not responding:**
- Show warning banner at top
- "Daemon not running. Click to start."

### Validation Errors

**Settings:**
- Inline validation on blur
- Red border + error message below field
- Clear error when user starts typing

---

## Future Enhancements

**v2 Features:**
- Session tagging (local tags)
- Session notes (markdown)
- Export session as PDF/HTML
- Session comparison (diff two sessions)
- Dark mode auto-switch based on time

**v3 Features:**
- Plugin system for custom views
- AI-powered session summaries
- Session analytics (time spent, tools used)

---

## Component Library Reference

### Shadcn UI Components Used

- **Button:** Primary, secondary, ghost variants
- **Card:** Session cards, tool cards
- **Input:** Search, settings
- **Select:** Dropdown filters
- **Dialog:** Modals (share, errors)
- **Badge:** Tool badges, status indicators
- **Separator:** Horizontal rules
- **Tooltip:** Hover explanations
- **ScrollArea:** For long content

### Custom Components

**SessionCard:**
```tsx
<SessionCard
  session={session}
  onClick={() => navigate(`/sessions/${session.id}`)}
  onShare={() => openShareModal(session)}
/>
```

**MessageBubble:**
```tsx
<MessageBubble
  role="user" | "assistant"
  content={content}
  timestamp={timestamp}
/>
```

**ToolCard:**
```tsx
<ToolCard
  toolName="write"
  input={input}
  output={output}
  isError={false}
/>
```

**ThinkingBlock:**
```tsx
<ThinkingBlock
  content={thinking}
  defaultExpanded={false}
/>
```

---

## Conclusion

This local UI provides:
- ✅ **Clean design** - Minimal, focused on content
- ✅ **Fast** - Direct filesystem reads, optimized rendering
- ✅ **Delightful** - Whimsical touches, smooth animations
- ✅ **Accessible** - Keyboard navigation, ARIA labels, screen reader support
- ✅ **Responsive** - Works on mobile, tablet, desktop
- ✅ **Search-first** - Powerful filtering and full-text search

**Next Steps:**
1. Generate Remote Server UX SPEC
2. Generate implementation SPECs (daemon, CLI, server)

---

**Document Status:** Ready for review
**Last Updated:** 2026-01-28
