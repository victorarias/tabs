const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

async function testTranscriptViewer() {
    const browser = await chromium.launch({ headless: false });
    const context = await browser.newContext();
    const page = await context.newPage();

    // Load the transcript viewer
    const viewerPath = path.join(__dirname, 'transcript-viewer.html');
    await page.goto(`file://${viewerPath}`);

    console.log('✓ Opened transcript viewer');

    // Take initial screenshot
    await page.screenshot({ path: 'screenshot-01-initial.png', fullPage: true });
    console.log('✓ Screenshot: initial state (with demo)');

    // Read the transcript file
    const transcriptPath = '/Users/victor.arias/.claude/projects/-Users-victor-arias-projects-tabs2/6e2ff10e-3a89-4a0b-ad0f-1e365807dfe5.jsonl';
    const transcriptContent = fs.readFileSync(transcriptPath, 'utf-8');

    console.log('✓ Read transcript file');

    // Inject the transcript by simulating what handleDroppedFiles does
    await page.evaluate((content) => {
        // Parse JSONL
        const lines = content.trim().split('\n');
        const messages = [];
        for (const line of lines) {
            if (line.trim()) {
                try {
                    messages.push(JSON.parse(line));
                } catch (e) {
                    console.error('Failed to parse line:', e);
                }
            }
        }

        // Add to transcripts state
        window.transcripts = window.transcripts || {};
        window.transcripts['test-session.jsonl'] = messages;

        // Add to dropdown
        const fileSelect = document.getElementById('fileSelect');
        const option = document.createElement('option');
        option.value = 'test-session.jsonl';
        option.textContent = 'test-session.jsonl';
        fileSelect.appendChild(option);
        fileSelect.disabled = false;
        fileSelect.value = 'test-session.jsonl';

        // Trigger display
        if (typeof displayTranscript === 'function') {
            displayTranscript('test-session.jsonl');
        }
    }, transcriptContent);

    console.log('✓ Injected transcript data');

    // Wait for rendering
    await page.waitForTimeout(1000);

    // Take screenshot of loaded transcript
    await page.screenshot({ path: 'screenshot-02-loaded.png', fullPage: true });
    console.log('✓ Screenshot: transcript loaded');

    // Check for messages
    const messageCount = await page.locator('.message').count();
    console.log(`✓ Found ${messageCount} messages rendered`);

    // Check for tool calls
    const toolCallCount = await page.locator('.tool-call').count();
    console.log(`✓ Found ${toolCallCount} tool calls rendered`);

    // Check for thinking blocks
    const thinkingCount = await page.locator('.thinking-block').count();
    console.log(`✓ Found ${thinkingCount} thinking blocks rendered`);

    // Expand a thinking block if present
    const thinkingHeaders = page.locator('.thinking-block .collapsible-header');
    if (await thinkingHeaders.count() > 0) {
        await thinkingHeaders.first().click();
        await page.waitForTimeout(300);
        await page.screenshot({ path: 'screenshot-03-thinking-expanded.png', fullPage: true });
        console.log('✓ Screenshot: thinking block expanded');
    }

    // Expand a tool call if present
    const toolHeaders = page.locator('.tool-call .collapsible-header');
    if (await toolHeaders.count() > 0) {
        await toolHeaders.first().click();
        await page.waitForTimeout(300);
        await page.screenshot({ path: 'screenshot-04-tool-expanded.png', fullPage: true });
        console.log('✓ Screenshot: tool call expanded');
    }

    // Scroll to bottom
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(300);
    await page.screenshot({ path: 'screenshot-05-bottom.png', fullPage: false });
    console.log('✓ Screenshot: scrolled to bottom');

    // Test drag overlay appears (simulate dragover)
    await page.evaluate(() => {
        const event = new DragEvent('dragover', {
            bubbles: true,
            cancelable: true,
            dataTransfer: new DataTransfer()
        });
        document.dispatchEvent(event);
    });
    await page.waitForTimeout(300);
    await page.screenshot({ path: 'screenshot-06-drag-overlay.png' });
    console.log('✓ Screenshot: drag overlay');

    // Hide overlay
    await page.evaluate(() => {
        document.getElementById('dropOverlay').classList.remove('show');
    });

    // Test Finder button (toast)
    await page.click('button:has-text("Finder")');
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'screenshot-07-toast.png' });
    console.log('✓ Screenshot: toast notification');

    console.log('\n=== Test Summary ===');
    console.log(`Messages: ${messageCount}`);
    console.log(`Tool calls: ${toolCallCount}`);
    console.log(`Thinking blocks: ${thinkingCount}`);
    console.log('Screenshots saved to current directory');

    // Keep browser open for manual inspection
    console.log('\nBrowser left open for inspection. Press Ctrl+C to close.');

    // Wait indefinitely (user closes manually)
    await new Promise(() => {});
}

testTranscriptViewer().catch(console.error);
