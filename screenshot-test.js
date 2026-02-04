const { chromium } = require('playwright');
const fs = require('fs');
const http = require('http');
const path = require('path');

async function main() {
    const server = http.createServer((req, res) => {
        const content = fs.readFileSync(path.join(__dirname, 'transcript-viewer.html'), 'utf-8');
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(content);
    }).listen(3334);

    const browser = await chromium.launch();
    const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
    await page.goto('http://localhost:3334');

    const transcript = fs.readFileSync('/Users/victor.arias/.claude/projects/-Users-victor-arias-projects-tabs2/6e2ff10e-3a89-4a0b-ad0f-1e365807dfe5.jsonl', 'utf-8');

    await page.evaluate((content) => {
        const lines = content.trim().split('\n');
        const messages = lines.map(l => { try { return JSON.parse(l); } catch(e) { return null; } }).filter(Boolean);
        transcripts['session.jsonl'] = messages;
        const sel = document.getElementById('fileSelect');
        const opt = document.createElement('option');
        opt.value = 'session.jsonl';
        opt.textContent = 'session.jsonl';
        sel.appendChild(opt);
        sel.disabled = false;
        sel.value = 'session.jsonl';
        renderTranscript('session.jsonl');
    }, transcript);

    await page.waitForTimeout(500);
    await page.screenshot({ path: 'final-test.png', fullPage: false });
    console.log('Screenshot saved: final-test.png');

    await browser.close();
    server.close();
}
main();
