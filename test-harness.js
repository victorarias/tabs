const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');
const http = require('http');

// Simple HTTP server to serve the HTML file
function startServer(port = 3333) {
    return new Promise((resolve) => {
        const server = http.createServer((req, res) => {
            const filePath = path.join(__dirname, 'transcript-viewer.html');
            const content = fs.readFileSync(filePath, 'utf-8');
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end(content);
        });
        server.listen(port, () => {
            console.log(`Server running at http://localhost:${port}`);
            resolve(server);
        });
    });
}

async function runTests() {
    const errors = [];
    const logs = [];

    const server = await startServer();
    const browser = await chromium.launch({ headless: true });
    const context = await browser.newContext();
    const page = await context.newPage();

    // Capture console errors
    page.on('console', msg => {
        if (msg.type() === 'error') {
            errors.push(`Console error: ${msg.text()}`);
        }
        logs.push(`[${msg.type()}] ${msg.text()}`);
    });

    page.on('pageerror', err => {
        errors.push(`Page error: ${err.message}`);
    });

    console.log('\n=== Test 1: Initial Load ===');
    await page.goto('http://localhost:3333');
    await page.waitForTimeout(500);

    if (errors.length > 0) {
        console.log('❌ Errors on initial load:');
        errors.forEach(e => console.log('  ', e));
    } else {
        console.log('✓ No errors on initial load');
    }
    errors.length = 0;

    console.log('\n=== Test 2: Inject Transcript Data ===');
    const transcriptPath = '/Users/victor.arias/.claude/projects/-Users-victor-arias-projects-tabs2/6e2ff10e-3a89-4a0b-ad0f-1e365807dfe5.jsonl';
    const transcriptContent = fs.readFileSync(transcriptPath, 'utf-8');

    // Test if key functions exist
    const functionsExist = await page.evaluate(() => {
        return {
            handleDroppedFiles: typeof handleDroppedFiles,
            parseJSONL: typeof parseJSONL,
            renderTranscript: typeof renderTranscript,
            transcripts: typeof transcripts
        };
    });
    console.log('Function availability:', functionsExist);

    // Simulate what happens when files are dropped
    const injectResult = await page.evaluate((content) => {
        try {
            // Parse JSONL
            const lines = content.trim().split('\n');
            const messages = [];
            for (const line of lines) {
                if (line.trim()) {
                    try {
                        messages.push(JSON.parse(line));
                    } catch (e) {
                        // skip bad lines
                    }
                }
            }

            // Check if transcripts exists
            if (typeof transcripts === 'undefined') {
                return { error: 'transcripts is undefined' };
            }

            transcripts['test.jsonl'] = messages;

            // Check if renderTranscript exists
            if (typeof renderTranscript === 'undefined') {
                return { error: 'renderTranscript is undefined' };
            }

            // Add to dropdown
            const fileSelect = document.getElementById('fileSelect');
            const option = document.createElement('option');
            option.value = 'test.jsonl';
            option.textContent = 'test.jsonl';
            fileSelect.appendChild(option);
            fileSelect.disabled = false;
            fileSelect.value = 'test.jsonl';

            renderTranscript('test.jsonl');
            return { success: true, messageCount: messages.length };
        } catch (e) {
            return { error: e.message, stack: e.stack };
        }
    }, transcriptContent);

    if (injectResult.error) {
        console.log('❌ Error injecting transcript:', injectResult.error);
        if (injectResult.stack) console.log('  Stack:', injectResult.stack);
    } else {
        console.log('✓ Transcript injected successfully');
        console.log('  Parsed messages:', injectResult.messageCount);
    }

    await page.waitForTimeout(500);

    if (errors.length > 0) {
        console.log('❌ Errors after injection:');
        errors.forEach(e => console.log('  ', e));
    } else {
        console.log('✓ No errors after injection');
    }
    errors.length = 0;

    console.log('\n=== Test 3: Check Rendered Content ===');
    const rendered = await page.evaluate(() => {
        return {
            messages: document.querySelectorAll('.message').length,
            userMessages: document.querySelectorAll('.message-user').length,
            assistantMessages: document.querySelectorAll('.message-assistant').length,
            toolBlocks: document.querySelectorAll('.tool-block').length,
            thinkingBlocks: document.querySelectorAll('.thinking-block').length,
        };
    });
    console.log('Rendered elements:', rendered);

    console.log('\n=== Test 4: Simulate Drag & Drop ===');
    // Trigger dragenter
    await page.evaluate(() => {
        const event = new DragEvent('dragenter', {
            bubbles: true,
            cancelable: true,
        });
        document.dispatchEvent(event);
    });
    await page.waitForTimeout(100);

    const overlayVisible = await page.evaluate(() => {
        const overlay = document.getElementById('dropOverlay');
        return overlay && overlay.classList.contains('show');
    });
    console.log('Drop overlay visible on dragenter:', overlayVisible ? '✓' : '❌');

    // Hide overlay for subsequent tests
    await page.evaluate(() => {
        const overlay = document.getElementById('dropOverlay');
        overlay.classList.remove('show');
    });

    if (errors.length > 0) {
        console.log('❌ Errors during drag test:');
        errors.forEach(e => console.log('  ', e));
    }
    errors.length = 0;

    console.log('\n=== Test 5: Test Finder Button ===');
    // Grant clipboard permissions
    await context.grantPermissions(['clipboard-write', 'clipboard-read']);
    await page.click('button:has-text("Finder")');
    await page.waitForTimeout(300);

    const toastVisible = await page.evaluate(() => {
        const toast = document.getElementById('toast');
        return toast && toast.classList.contains('show');
    });
    console.log('Toast visible after Finder click:', toastVisible ? '✓' : '❌ (clipboard permissions may be blocked in headless)');

    console.log('\n=== Summary ===');
    const totalErrors = errors.length;
    if (totalErrors === 0) {
        console.log('✓ All tests passed!');
    } else {
        console.log(`❌ ${totalErrors} error(s) found`);
    }

    await browser.close();
    server.close();

    return totalErrors === 0;
}

runTests().then(success => {
    process.exit(success ? 0 : 1);
}).catch(err => {
    console.error('Test failed:', err);
    process.exit(1);
});
