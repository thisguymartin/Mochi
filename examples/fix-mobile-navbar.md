# Fix Mobile Navbar

The mobile navigation menu has multiple issues reported by QA.

## Bug Report

**Environment:** iOS Safari 17, Android Chrome 120, viewport < 768px

### Issue 1: Menu doesn't close on route change
When the user taps a nav link and the page navigates, the hamburger menu stays open,
overlaying the new page content. Expected behavior: menu closes on navigation.

### Issue 2: Menu items not tappable near edges
The first and last menu items have insufficient tap targets. Users report needing
multiple taps to activate them. The touch target should be at least 44x44px per
WCAG 2.5.5.

### Issue 3: Scroll lock not applied
When the menu is open, the background page is still scrollable. Opening the menu
should apply `overflow: hidden` to the body.

## Reproduction Steps

1. Open the app on a mobile device (or use DevTools responsive mode at 375px width)
2. Tap the hamburger icon to open the menu
3. Tap any navigation link
4. Observe: menu remains open after navigation

## Files Likely Involved

- `src/components/Navbar.tsx`
- `src/components/Navbar.module.css` or equivalent styles
- `src/hooks/useMediaQuery.ts` (if breakpoint detection is relevant)
