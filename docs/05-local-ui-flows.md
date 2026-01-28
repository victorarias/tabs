# Local UI Flows Specification: tabs

**Version:** 2.0
**Date:** 2026-01-28
**Status:** SPEC

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

## Design Philosophy: "Sheet Music for AI"

tabs isn't just another transcript viewer—it's a **tablature reader for AI conversations**. Like guitar tabs share the fingerwork behind a song, tabs shares the prompts and reasoning behind great code.

**Core Aesthetic:** Editorial minimalism with musical details. Clean layouts inspired by sheet music, music software, and vinyl record liner notes.

**Memorable Element:** Musical notation-inspired micro-interactions and typography that feels like reading beautifully typeset sheet music.

---

## Visual Design Language

### Typography System

**Display Font (Headings, UI Chrome):**
- **Primary:** [DM Serif Display](https://fonts.google.com/specimen/DM+Serif+Display)
- **Fallback:** Georgia, serif
- **Usage:** Page titles, session titles, modal headers
- **Why:** Evokes printed music scores and vintage record sleeves

**Body Font (Messages, Content):**
- **Primary:** [IBM Plex Sans](https://fonts.google.com/specimen/IBM+Plex+Sans)
- **Fallback:** -apple-system, system-ui, sans-serif
- **Usage:** Message text, descriptions, body copy
- **Why:** Technical precision meets approachability

**Code Font (Tool Interactions, JSON):**
- **Primary:** [JetBrains Mono](https://fonts.google.com/specimen/JetBrains+Mono)
- **Fallback:** 'Fira Code', 'Monaco', 'Consolas', monospace
- **Usage:** Code blocks, API responses, tool parameters

**Sizes (Fluid Typography):**
```css
--font-display: 'DM Serif Display', Georgia, serif;
--font-sans: 'IBM Plex Sans', system-ui, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;

/* Display */
--text-display: clamp(2rem, 5vw, 3.5rem);      /* 32-56px */
--text-h1: clamp(1.75rem, 4vw, 2.5rem);        /* 28-40px */
--text-h2: clamp(1.5rem, 3vw, 2rem);           /* 24-32px */
--text-h3: clamp(1.25rem, 2.5vw, 1.75rem);     /* 20-28px */

/* Body */
--text-base: 1rem;           /* 16px */
--text-sm: 0.875rem;         /* 14px */
--text-xs: 0.75rem;          /* 12px */
--text-lg: 1.125rem;         /* 18px */

/* Line Heights */
--leading-tight: 1.2;
--leading-normal: 1.5;
--leading-relaxed: 1.75;
```

### Color Palette: "Vinyl & Mahogany"

**Inspiration:** Vintage audio equipment, warm studio lighting, aged paper

**Light Theme (Default):**
```css
/* Base */
--bg-primary: #faf9f7;        /* Warm off-white, like aged paper */
--bg-secondary: #f0ede8;      /* Subtle cream */
--fg-primary: #1a1614;        /* Rich near-black */
--fg-secondary: #524c47;      /* Warm gray */
--fg-tertiary: #857d77;       /* Muted taupe */

/* Accent: Amber (like warm studio lighting) */
--accent: #d97706;            /* Rich amber */
--accent-hover: #b45309;      /* Deeper amber */
--accent-subtle: #fef3c7;     /* Light amber wash */

/* Semantic */
--border: #e0dbd5;            /* Soft divider */
--success: #059669;           /* Emerald */
--error: #dc2626;             /* Red */
--warning: #d97706;           /* Amber */

/* Code Highlighting (warm, muted tones) */
--syntax-bg: #f5f2ed;
--syntax-comment: #9ca3af;
--syntax-keyword: #b45309;    /* Amber */
--syntax-string: #059669;     /* Emerald */
--syntax-function: #2563eb;   /* Blue */
```

**Dark Theme: "Night Studio"**
```css
/* Base */
--bg-primary: #0f0e0d;        /* Deep warm black */
--bg-secondary: #1a1816;      /* Slightly lighter */
--fg-primary: #f5f2ed;        /* Warm off-white */
--fg-secondary: #c7c0b8;      /* Warm light gray */
--fg-tertiary: #857d77;       /* Muted taupe */

/* Accent: Warm Gold */
--accent: #f59e0b;            /* Gold */
--accent-hover: #fbbf24;      /* Brighter gold */
--accent-subtle: #78350f;     /* Dark amber */

/* Semantic */
--border: #2d2a27;
--success: #10b981;
--error: #f87171;
--warning: #fbbf24;

/* Code Highlighting */
--syntax-bg: #1a1816;
--syntax-comment: #6b7280;
--syntax-keyword: #fbbf24;
--syntax-string: #34d399;
--syntax-function: #60a5fa;
```

### Spacing & Layout: "Staff Lines"

**Grid System:** 8px base unit (like musical measures)

```css
--space-1: 0.25rem;   /* 4px */
--space-2: 0.5rem;    /* 8px - base unit */
--space-3: 0.75rem;   /* 12px */
--space-4: 1rem;      /* 16px */
--space-6: 1.5rem;    /* 24px */
--space-8: 2rem;      /* 32px */
--space-12: 3rem;     /* 48px */
--space-16: 4rem;     /* 64px */
--space-24: 6rem;     /* 96px */

/* Container Widths */
--container-sm: 640px;
--container-md: 768px;
--container-lg: 1024px;
--container-xl: 1280px;
--container-content: 65ch;  /* Optimal reading width */
```

### Musical Micro-Interactions

**1. Note Animations (Staggered Reveals):**
Timeline items fade in with slight vertical slide, staggered by 40ms like notes being played in sequence.

```css
@keyframes noteIn {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.timeline-item {
  animation: noteIn 300ms cubic-bezier(0.16, 1, 0.3, 1);
  animation-fill-mode: backwards;
}

.timeline-item:nth-child(1) { animation-delay: 0ms; }
.timeline-item:nth-child(2) { animation-delay: 40ms; }
.timeline-item:nth-child(3) { animation-delay: 80ms; }
```

**2. Hover States (Crescendo Effect):**
Cards and buttons scale slightly and lift with shadow on hover.

```css
.session-card {
  transition: transform 200ms cubic-bezier(0.16, 1, 0.3, 1),
              box-shadow 200ms cubic-bezier(0.16, 1, 0.3, 1);
}

.session-card:hover {
  transform: translateY(-2px) scale(1.005);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
```

**3. Click Feedback (Staccato):**
Brief scale-down on click before navigation.

```css
.session-card:active {
  transform: scale(0.98);
  transition-duration: 100ms;
}
```

**4. Loading States (Metronome Pulse):**
Subtle pulsing animation for loading indicators.

```css
@keyframes metronome {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}

.loading {
  animation: metronome 1.2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
```

**5. Search Input (Focus Crescendo):**
When search is focused, border color transitions smoothly and a subtle glow appears.

```css
.search-input {
  transition: border-color 200ms, box-shadow 200ms;
}

.search-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}
```

### Musical Details: Custom Icons

**Guitar Pick Icon (instead of generic dots):**
```svg
<svg viewBox="0 0 24 24" fill="currentColor">
  <path d="M12 2L8 8L6 12L8 16L12 22L16 16L18 12L16 8L12 2Z" opacity="0.9"/>
</svg>
```

**Metronome Icon (for loading):**
```svg
<svg viewBox="0 0 24 24" fill="currentColor">
  <path d="M12 4L8 20H16L12 4Z"/>
  <circle cx="12" cy="10" r="2"/>
</svg>
```

### Tailwind Configuration

```js
// tailwind.config.js
module.exports = {
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        display: ['DM Serif Display', 'Georgia', 'serif'],
        sans: ['IBM Plex Sans', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      colors: {
        bg: {
          primary: 'var(--bg-primary)',
          secondary: 'var(--bg-secondary)',
        },
        fg: {
          primary: 'var(--fg-primary)',
          secondary: 'var(--fg-secondary)',
          tertiary: 'var(--fg-tertiary)',
        },
        accent: {
          DEFAULT: 'var(--accent)',
          hover: 'var(--accent-hover)',
          subtle: 'var(--accent-subtle)',
        },
        border: 'var(--border)',
      },
      animation: {
        'note-in': 'noteIn 300ms cubic-bezier(0.16, 1, 0.3, 1)',
        'metronome': 'metronome 1.2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        noteIn: {
          from: { opacity: 0, transform: 'translateY(12px)' },
          to: { opacity: 1, transform: 'translateY(0)' },
        },
        metronome: {
          '0%, 100%': { opacity: 0.3 },
          '50%': { opacity: 1 },
        },
      },
    },
  },
};
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
- Logo: "tabs" with guitar pick icon (custom SVG)
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

### Music Theme Details

**Logo:**
- Custom guitar pick SVG icon (see Visual Design Language section)
- Subtle rotation on hover (3deg)
- DM Serif Display font for "tabs" wordmark

**Loading States:**
- Metronome pulse animation (see Musical Micro-Interactions)
- Rotating guitar pick for longer operations

**Success Animations:**
- Share success: Brief scale pulse (200ms)
- Copy success: Checkmark with bounce

**Empty States:**
- Animated guitar icon that "strums" on hover
- Warm, friendly copy

### Musical Micro-interactions Summary

**Staggered Reveals:**
- Timeline items: 40ms delay per item (like notes being played)

**Crescendo Effects:**
- Hover states: lift + shadow grow
- Focus states: glow effect

**Staccato Clicks:**
- Brief scale-down on click (0.98)
- 100ms transition

**Metronome Pulse:**
- Loading states pulse at 1.2s intervals

See "Musical Micro-Interactions" section above for implementation details.

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
