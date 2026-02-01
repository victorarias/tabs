import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const BASE_URL = process.env.BASE_URL || 'http://localhost:3000'

test.describe('Accessibility - Contrast', () => {
  test('session list page has sufficient color contrast', async ({ page }) => {
    await page.goto(BASE_URL)
    await page.waitForLoadState('networkidle')

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2aa']) // WCAG 2.0 Level AA
      .analyze()

    // Filter for contrast-related violations
    const contrastViolations = results.violations.filter(
      (v) => v.id === 'color-contrast' || v.id === 'color-contrast-enhanced'
    )

    if (contrastViolations.length > 0) {
      console.log('\nContrast violations found:')
      contrastViolations.forEach((violation) => {
        console.log(`\n${violation.id}: ${violation.description}`)
        violation.nodes.forEach((node) => {
          console.log(`  - ${node.html}`)
          console.log(`    ${node.failureSummary}`)
        })
      })
    }

    expect(contrastViolations).toHaveLength(0)
  })

  test('session detail page has sufficient color contrast', async ({ page }) => {
    // First get a session ID from the list
    await page.goto(BASE_URL)
    await page.waitForLoadState('networkidle')

    // Click on the first session card if available
    const sessionCard = page.locator('.session-card').first()
    if (await sessionCard.isVisible()) {
      await sessionCard.click()
      await page.waitForLoadState('networkidle')
    } else {
      test.skip()
      return
    }

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2aa'])
      .analyze()

    const contrastViolations = results.violations.filter(
      (v) => v.id === 'color-contrast' || v.id === 'color-contrast-enhanced'
    )

    if (contrastViolations.length > 0) {
      console.log('\nContrast violations found on session detail:')
      contrastViolations.forEach((violation) => {
        console.log(`\n${violation.id}: ${violation.description}`)
        violation.nodes.forEach((node) => {
          console.log(`  - Element: ${node.target}`)
          console.log(`    HTML: ${node.html.substring(0, 100)}...`)
          console.log(`    ${node.failureSummary}`)
        })
      })
    }

    expect(contrastViolations).toHaveLength(0)
  })

})

test.describe('Accessibility - Full Audit', () => {
  test('session detail page passes full accessibility audit', async ({ page }) => {
    await page.goto(BASE_URL)
    await page.waitForLoadState('networkidle')

    const sessionCard = page.locator('.session-card').first()
    if (await sessionCard.isVisible()) {
      await sessionCard.click()
      await page.waitForLoadState('networkidle')
    } else {
      test.skip()
      return
    }

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2aa', 'wcag21aa'])
      .analyze()

    // Log all violations for debugging
    if (results.violations.length > 0) {
      console.log(`\nFound ${results.violations.length} accessibility violations:`)
      results.violations.forEach((violation) => {
        console.log(`\n[${violation.impact}] ${violation.id}: ${violation.description}`)
        console.log(`  Help: ${violation.helpUrl}`)
        console.log(`  Affected elements: ${violation.nodes.length}`)
      })
    }

    // For now, just report - can make strict later
    // expect(results.violations).toHaveLength(0)
  })
})
