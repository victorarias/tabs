# Frontend Design Brief: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** DESIGN BRIEF

---

## Design Philosophy: "Sheet Music for AI"

tabs isn't just another transcript viewer—it's a **tablature reader for AI conversations**. Like guitar tabs share the fingerwork behind a song, tabs shares the prompts and reasoning behind great code.

### Conceptual Direction: Editorial Minimalism with Musical Details

**Core Aesthetic:** Refined, editorial interface inspired by sheet music, music software, and vinyl record liner notes. Clean layouts with unexpected musical flourishes.

**Differentiation:** The one thing users will remember—**musical notation-inspired micro-interactions** and a typography system that feels like reading beautifully typeset sheet music.

---

## Visual Language

### Typography System

**Display Font (Headings, UI Chrome):**
- **Primary Choice:** [DM Serif Display](https://fonts.google.com/specimen/DM+Serif+Display) - elegant serif with musical character
- **Fallback:** Georgia, serif
- **Usage:** Page titles, session titles, modal headers
- **Why:** Evokes printed music scores and vintage record sleeves without being overly decorative

**Body Font (Messages, Content):**
- **Primary Choice:** [IBM Plex Sans](https://fonts.google.com/specimen/IBM+Plex+Sans) - neutral but warm, excellent legibility
- **Fallback:** -apple-system, system-ui, sans-serif
- **Usage:** Message text, descriptions, body copy
- **Why:** Technical precision meets approachability; pairs beautifully with DM Serif Display

**Code Font (Tool Interactions, JSON):**
- **Primary Choice:** [JetBrains Mono](https://fonts.google.com/specimen/JetBrains+Mono) - ligatures, excellent code rendering
- **Fallback:** 'Fira Code', 'Monaco', 'Consolas', monospace
- **Usage:** Code blocks, API responses, tool parameters
- **Why:** Industry-standard code font with character

**Sizes (Fluid Typography):**
```css
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

**Vertical Rhythm:** Use multiples of 8px for consistent spacing that feels like musical staff lines.

### Musical Micro-Interactions

**1. Note Animations (Staggered Reveals):**
When timeline items load, they fade in with a slight vertical slide, staggered by 40ms like notes being played in sequence.

```css
/* Timeline item entrance */
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
/* etc */
```

**2. Hover States (Crescendo Effect):**
Cards and buttons scale slightly and lift with shadow on hover, like a note being emphasized.

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
Brief scale-down on click before navigation, like tapping a key.

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

---

## Component Design

### Session Card (Timeline Item)

**Visual Treatment:**
- Rounded corners (12px) with subtle border
- White background (light) / elevated dark background
- Hover state: lift + shadow
- Left accent border (4px) in amber to indicate active/recent sessions

**Layout:**
```
┌─────────────────────────────────────┐
│ [Guitar Pick Icon] Session Title    │ <- DM Serif Display, 1.25rem
│ 2 hours ago • 47 messages           │ <- IBM Plex Sans, 0.875rem, muted
│                                     │
│ "Implement authentication system…" │ <- First message preview, italic
└─────────────────────────────────────┘
```

**Musical Detail:** Guitar pick icon (custom SVG) instead of generic dot/circle

### Session Detail Layout

**Structure:** Two-column layout on desktop, single column mobile

```
┌──────────────────────────────────────────────────────┐
│ [Header]                                             │
│ ← Back to Timeline    Session Title    [Share] [⚙]   │
│                                                      │
│ ┌─────────────────┐ ┌──────────────────────────────┐│
│ │ Sidebar         │ │ Messages & Tools             ││
│ │                 │ │                              ││
│ │ • Overview      │ │ [Message Bubble]             ││
│ │ • 47 messages   │ │ [Tool Interaction]           ││
│ │ • 12 tools used │ │ [Message Bubble]             ││
│ │                 │ │ ...                          ││
│ │ [Timeline Mini] │ │                              ││
│ └─────────────────┘ └──────────────────────────────┘│
└──────────────────────────────────────────────────────┘
```

**Sidebar Musical Detail:** Mini vertical timeline with dot markers connected by vertical line (like a musical staff)

### Message Bubbles

**User Messages:**
- Right-aligned
- Background: accent color with 10% opacity
- Border-left: 3px solid accent color
- Rounded: 12px 12px 4px 12px

**Assistant Messages:**
- Left-aligned
- Background: secondary background color
- Border-left: 3px solid muted border
- Rounded: 12px 12px 12px 4px

**Typography:**
- Message text: IBM Plex Sans, 1rem, leading-relaxed
- Timestamp: 0.75rem, muted, monospace

### Tool Interaction Blocks

**Visual Treatment:**
- Collapsible accordion
- Background: subtle gradient (light amber → transparent)
- Icon: Tool-specific (Bash, Read, Write, etc.) from Lucide
- Code blocks: syntax highlighted with warm color scheme

**Layout:**
```
┌──────────────────────────────────────┐
│ [⚡ Bash] Run tests               [▼]│
│                                      │
│ $ npm test                           │ <- Code block
│ ✓ All tests passed                   │ <- Result
│                                      │
│ 245ms                                │ <- Duration, muted
└──────────────────────────────────────┘
```

**Musical Detail:** Duration displayed in milliseconds with a subtle metronome icon

### Search Bar

**Position:** Fixed at top of timeline view, sticky on scroll

**Visual Treatment:**
- Full-width on mobile, max-width 640px on desktop
- Large input (48px height) with rounded corners (24px)
- Search icon inside left, keyboard shortcut hint inside right
- Glass morphism: `backdrop-filter: blur(8px)` with semi-transparent background

**Interaction:**
- Focus: Expand slightly + shadow
- Typing: Show live results count below
- Clear button appears when text entered

**Musical Detail:** Keyboard shortcut shown as "⌘K" or "Ctrl+K" in a subtle "key cap" style badge

### Share Modal

**Visual Treatment:**
- Center modal with dramatic backdrop blur
- Large header with share icon
- Form fields for tags, description
- Preview of what will be shared
- Prominent "Share with Team" button

**Layout:**
```
┌─────────────────────────────────────┐
│          Share Session              │
│  [🎸 Guitar icon]                   │
│                                     │
│  Session: "Implement auth system"   │
│  47 messages • 2 hours ago          │
│                                     │
│  Tags: [frontend] [auth] [+]        │
│  Description: ___________________   │
│                                     │
│         [Share with Team]           │
│              [Cancel]               │
└─────────────────────────────────────┘
```

**Animation:** Modal scales in from 0.95 → 1.0 with fade, 300ms

---

## Page-Specific Layouts

### Timeline View (Homepage)

**Header:**
- Logo + "tabs" wordmark (DM Serif Display)
- Search bar (prominent)
- Settings icon (top right)
- Theme toggle (sun/moon icon)

**Day Groups:**
- Date headers: DM Serif Display, 1.5rem, with subtle divider line
- "Today", "Yesterday", then "January 28, 2026" format
- Sessions within each day ordered newest first

**Empty State:**
```
┌─────────────────────────────────────┐
│                                     │
│        [🎸 Large guitar icon]       │
│                                     │
│      No sessions yet                │
│   Start a conversation in           │
│   Claude Code to see it here        │
│                                     │
└─────────────────────────────────────┘
```

**Musical Detail:** Empty state icon could be an animated guitar that "strums" on hover

### Session Detail View

**Header:**
- Back button with smooth page transition
- Session title (editable on hover → pencil icon)
- Metadata: timestamp, message count, tool count
- Actions: Share, Settings

**Message Flow:**
- Chronological, top to bottom
- Time gaps shown with subtle divider + timestamp
- Scroll to bottom button when not at bottom

**Sidebar (Desktop only):**
- Session stats
- Quick navigation (jump to tool use, jump to errors)
- Mini timeline with clickable dots

**Musical Detail:** Mini timeline dots could be styled like notes on a staff line

### Settings Page

**Layout:**
- Simple vertical form
- Sections: Appearance, Storage, Privacy, About
- Toggle switches for dark mode, compact view, etc.
- Path display for local storage location

**Visual Treatment:**
- Clean, spacious (24px between sections)
- Section headers: DM Serif Display
- Form labels: IBM Plex Sans, 0.875rem

---

## Responsive Breakpoints

```css
/* Mobile first approach */
--breakpoint-sm: 640px;   /* Landscape phones */
--breakpoint-md: 768px;   /* Tablets */
--breakpoint-lg: 1024px;  /* Desktops */
--breakpoint-xl: 1280px;  /* Large desktops */
```

**Mobile (<768px):**
- Single column layout
- Hamburger menu for navigation
- Search bar full width
- Session cards stack vertically
- Hide sidebar, show as bottom sheet if needed

**Tablet (768px - 1024px):**
- Two-column grid for session cards
- Sidebar optional, can toggle
- Search bar centered, max-width

**Desktop (>1024px):**
- Three-column grid for session cards (optional)
- Sidebar always visible
- Generous padding and spacing

---

## Dark Mode Strategy

**Implementation:** CSS custom properties with class toggle on `<html>` element

```html
<html class="dark">
```

**Transition:** Smooth color transitions on theme change

```css
* {
  transition: background-color 200ms, color 200ms, border-color 200ms;
}
```

**Persistence:** Save preference to `localStorage`

**Musical Detail:** Theme toggle could be a sun/moon icon that "slides" like a volume fader

---

## Accessibility

**Focus States:**
- Clear focus ring (3px solid accent color, 3px offset)
- Skip to content link
- Keyboard navigation for all interactive elements

**Semantic HTML:**
- Proper heading hierarchy
- `<main>`, `<nav>`, `<article>` landmarks
- ARIA labels for icon buttons

**Color Contrast:**
- All text meets WCAG AA (4.5:1 for body, 3:1 for large text)
- Dark mode also meets contrast requirements

**Motion:**
- Respect `prefers-reduced-motion`

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## Performance Constraints

**Bundle Size:**
- Keep total JS under 200KB (gzipped)
- Lazy load Framer Motion for non-critical animations
- Code syntax highlighting: lazy load Shiki

**Loading Strategy:**
- SSR for initial page load (TanStack Start)
- Skeleton screens for loading states
- Virtualized lists for long timelines (react-window or TanStack Virtual)

**Image Optimization:**
- Use SVG for icons
- No raster images except user avatars (if added later)

---

## Animation Timing

**Principles:**
- Fast enough to feel snappy (200-300ms)
- Easing: `cubic-bezier(0.16, 1, 0.3, 1)` for most (smooth ease-out)
- Stagger delays: 40ms per item for sequential reveals
- Hover responses: 200ms
- Modal opens: 300ms
- Page transitions: 400ms

**Implementation:**
```js
// Framer Motion variants
const staggerContainer = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: {
      staggerChildren: 0.04,
    },
  },
};

const item = {
  hidden: { opacity: 0, y: 12 },
  show: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.3,
      ease: [0.16, 1, 0.3, 1],
    },
  },
};
```

---

## Code Example: Session Card Component

```tsx
import { motion } from 'framer-motion';
import { GuitarPick } from './icons/GuitarPick'; // Custom SVG icon

interface SessionCardProps {
  id: string;
  title: string;
  timestamp: string;
  messageCount: number;
  preview: string;
  isRecent?: boolean;
}

export function SessionCard({
  id,
  title,
  timestamp,
  messageCount,
  preview,
  isRecent = false,
}: SessionCardProps) {
  return (
    <motion.a
      href={`/sessions/${id}`}
      className={`
        group block rounded-xl border p-6
        transition-all duration-200 ease-out
        hover:shadow-lg hover:-translate-y-0.5 hover:scale-[1.005]
        active:scale-[0.98]
        ${isRecent ? 'border-l-4 border-l-accent' : 'border-border'}
        bg-bg-primary
      `}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
    >
      <div className="flex items-start gap-3">
        <GuitarPick className="mt-1 size-5 text-accent opacity-60 transition-opacity group-hover:opacity-100" />
        <div className="flex-1 space-y-2">
          <h3 className="font-display text-xl leading-tight text-fg-primary">
            {title}
          </h3>
          <div className="flex items-center gap-2 text-sm text-fg-tertiary">
            <time>{timestamp}</time>
            <span>•</span>
            <span>{messageCount} messages</span>
          </div>
          {preview && (
            <p className="italic text-sm text-fg-secondary leading-relaxed line-clamp-2">
              "{preview}"
            </p>
          )}
        </div>
      </div>
    </motion.a>
  );
}
```

---

## Tailwind Configuration

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

## Implementation Checklist

### Phase 1: Foundation
- [ ] Set up TanStack Start project
- [ ] Install dependencies (Tailwind, Shadcn, Framer Motion, Lucide)
- [ ] Configure Tailwind with custom theme
- [ ] Load Google Fonts (DM Serif Display, IBM Plex Sans, JetBrains Mono)
- [ ] Set up CSS custom properties for light/dark themes
- [ ] Implement theme toggle with localStorage persistence

### Phase 2: Core Components
- [ ] SessionCard component with hover animations
- [ ] MessageBubble component (user/assistant variants)
- [ ] ToolInteraction component with accordion
- [ ] SearchBar component with focus states
- [ ] EmptyState component with guitar icon
- [ ] LoadingState component with metronome animation

### Phase 3: Pages
- [ ] Timeline view with day grouping
- [ ] Session detail view with sidebar
- [ ] Settings page
- [ ] 404 page (with musical empty state)

### Phase 4: Interactions
- [ ] Staggered timeline item reveals
- [ ] Page transitions
- [ ] Modal animations
- [ ] Keyboard shortcuts
- [ ] Scroll-to-top button

### Phase 5: Polish
- [ ] Dark mode refinements
- [ ] Responsive testing (mobile, tablet, desktop)
- [ ] Accessibility audit (focus states, ARIA, contrast)
- [ ] Performance optimization (lazy loading, code splitting)
- [ ] Playwright visual regression tests

---

## Musical Details: Custom Icons

Create custom SVG icons with musical character:

**Guitar Pick Icon:**
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

**Staff Line Divider:**
```svg
<svg viewBox="0 0 100 20" className="w-full h-5">
  <line x1="0" y1="5" x2="100" y2="5" stroke="currentColor" opacity="0.1"/>
  <line x1="0" y1="10" x2="100" y2="10" stroke="currentColor" opacity="0.2"/>
  <line x1="0" y1="15" x2="100" y2="15" stroke="currentColor" opacity="0.1"/>
</svg>
```

---

## Remote UI Differences

The remote UI shares the same design system but with these adjustments:

**Color Accent:** Use a different accent color (emerald instead of amber) to visually distinguish remote from local

```css
/* Remote UI accent */
--accent: #059669;            /* Emerald */
--accent-hover: #047857;      /* Deeper emerald */
--accent-subtle: #d1fae5;     /* Light emerald wash */
```

**Additional Components:**
- Tag filter pills (rounded, clickable, multi-select)
- User avatar badges (showing who uploaded)
- Visibility indicator (public/team/private)

**Header Differences:**
- Logo links to browse page (not local timeline)
- User menu (if authenticated via IAP)
- Upload stats badge ("127 sessions shared")

---

## Summary: The Memorable Details

What makes this design unforgettable:

1. **Musical typography pairing** - DM Serif Display + IBM Plex Sans feels like sheet music
2. **Guitar pick icon** - unexpected, thematic, delightful
3. **Staff line spacing** - consistent 8px grid creates visual rhythm
4. **Warm color palette** - vinyl & mahogany, not generic blue/purple
5. **Staggered note animations** - timeline items reveal like notes being played
6. **Metronome loading states** - subtle musical pulse
7. **Editorial minimalism** - refined, professional, not bland

This isn't "yet another transcript viewer"—it's a **carefully designed instrument** for reading and sharing AI conversations.
