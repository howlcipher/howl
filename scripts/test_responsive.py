#!/usr/bin/env python3
"""
Automated Responsive Regression & Continuous QA Test Suite for Howl Ecosystem Hub
Canonical Location: scripts/test_responsive.py

Capabilities:
1. Local Hub Regression Suite (default):
   Spins up a local server for docs/ and validates:
   - Zero document-level horizontal overflow (scrollWidth <= clientWidth)
   - Zero uncontained element-level bounding box overflows (rect.right <= clientWidth + 1.5)
   - Accurate DOM hierarchy and section rendering across 9 viewports (320px - 1440px)
   - Zero console errors or uncaught exceptions
   - Interactive components: Theme toggle, Drawer open/close lifecycle + ARIA state, Copy buttons
2. Cross-Ecosystem Smoke Checks (--ecosystem):
   Probes the 7 live deployed Howl Pages sites for HTTP availability, clean initial load,
   zero uncaught JS errors, and mobile container containment without tight coupling.

Usage:
  python3 scripts/test_responsive.py            # Run local Hub responsive test suite
  python3 scripts/test_responsive.py --ecosystem # Run public ecosystem smoke tests
  python3 scripts/test_responsive.py --all       # Run both local Hub and ecosystem tests
"""

import sys
import os
import argparse
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
from playwright.sync_api import sync_playwright

DOCS_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'docs'))

VIEWPORTS = [
    {'name': '320px_small_phone', 'width': 320, 'height': 568},
    {'name': '360px_android_phone', 'width': 360, 'height': 640},
    {'name': '375px_iphone_standard', 'width': 375, 'height': 667},
    {'name': '390px_modern_iphone', 'width': 390, 'height': 844},
    {'name': '430px_large_phone', 'width': 430, 'height': 932},
    {'name': '768px_tablet_portrait', 'width': 768, 'height': 1024},
    {'name': '1024px_tablet_landscape', 'width': 1024, 'height': 768},
    {'name': '1280px_desktop', 'width': 1280, 'height': 800},
    {'name': '1440px_large_desktop', 'width': 1440, 'height': 900},
]

ECOSYSTEM_SITES = {
    'Howl Hub': 'https://howlcipher.github.io/howl/',
    'HowlFrame': 'https://howlcipher.github.io/howlframe/',
    'HowlPlane': 'https://howlcipher.github.io/howlplane/',
    'HowlNotes': 'https://howlcipher.github.io/howlnotes/',
    'HowlChangeOps': 'https://howlcipher.github.io/howlchangeops/',
    'HowlBoard': 'https://howlcipher.github.io/howlboard/',
    'HowlWriter': 'https://howlcipher.github.io/howlwriter/',
}

ELEMENT_OVERFLOW_DIAGNOSTIC_JS = """() => {
    const doc = document.documentElement;
    const cw = doc.clientWidth;
    const sw = doc.scrollWidth;
    const hasDocOverflow = sw > cw;
    
    const overflowingElements = [];
    document.querySelectorAll('*').forEach(el => {
        // Exclude intentionally offscreen components
        if (el.closest('#eco-drawer') || el.id === 'eco-drawer') return;
        if (el.closest('#eco-drawer-overlay') || el.id === 'eco-drawer-overlay') return;
        if (el.classList.contains('skip-link')) return;
        
        const style = window.getComputedStyle(el);
        if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return;
        
        const r = el.getBoundingClientRect();
        if (r.width === 0 || r.height === 0) return;
        
        // Check if bounding box exceeds viewport with 1.5px subpixel tolerance
        if (r.right > cw + 1.5 || r.left < -1.5) {
            // Check if element is inside an intentional horizontal scroll container
            const scrollContainer = el.closest('pre, .table-wrap, [style*="overflow-x: auto"], [style*="overflow-x: scroll"]');
            const isInsideScroll = scrollContainer !== null && scrollContainer !== el;
            
            overflowingElements.push({
                tag: el.tagName.toLowerCase(),
                id: el.id || '',
                className: typeof el.className === 'string' ? el.className : '',
                rectLeft: Math.round(r.left),
                rectRight: Math.round(r.right),
                rectWidth: Math.round(r.width),
                clientWidth: cw,
                isInsideScrollContainer: isInsideScroll,
                textSnippet: (el.innerText || el.textContent || '').slice(0, 50).replace(/\\s+/g, ' ').trim()
            });
        }
    });
    
    const uncontained = overflowingElements.filter(e => !e.isInsideScrollContainer);
    return {
        hasDocOverflow,
        scrollWidth: sw,
        clientWidth: cw,
        uncontainedOverflows: uncontained,
        totalOverflowCount: overflowingElements.length
    };
}"""

class QuietHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=DOCS_DIR, **kwargs)
        
    def log_message(self, format, *args):
        pass  # Suppress request logging during test runs

def start_server(port=8765):
    server = HTTPServer(('127.0.0.1', port), QuietHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server

def run_local_hub_tests(base_url, verbose=False):
    print(f"[*] Running local Hub responsive regression suite on {base_url}")
    total_checks = 0
    passed_checks = 0
    failed_checks = []

    with sync_playwright() as p:
        browser = p.chromium.launch()
        
        for vp in VIEWPORTS:
            w, h, name = vp['width'], vp['height'], vp['name']
            page = browser.new_page(viewport={'width': w, 'height': h})
            console_errors = []
            page.on('console', lambda msg: console_errors.append(msg.text) if msg.type == 'error' else None)
            
            try:
                page.goto(base_url, wait_until='networkidle')
                total_checks += 1
                
                # Check 1: Document and Element-Level Overflow Diagnostics
                diag = page.evaluate(ELEMENT_OVERFLOW_DIAGNOSTIC_JS)
                uncontained = diag['uncontainedOverflows']
                has_doc_overflow = diag['hasDocOverflow']
                
                if not has_doc_overflow and len(uncontained) == 0:
                    passed_checks += 1
                    status_str = f"PASS (SW={diag['scrollWidth']}, CW={diag['clientWidth']})"
                else:
                    status_str = f"FAIL (SW={diag['scrollWidth']} > CW={diag['clientWidth']})"
                    err_msg = [f"{name} ({w}px): Layout overflow detected!"]
                    if has_doc_overflow:
                        err_msg.append(f"  - Document scrollWidth ({diag['scrollWidth']}px) > clientWidth ({diag['clientWidth']}px)")
                    if uncontained:
                        err_msg.append("  - Offending uncontained elements:")
                        for u in uncontained:
                            selector = f"<{u['tag']}"
                            if u['id']:
                                selector += f" id=\"{u['id']}\""
                            if u['className']:
                                selector += f" class=\"{u['className']}\""
                            selector += ">"
                            err_msg.append(f"    * {selector} [left: {u['rectLeft']}px, right: {u['rectRight']}px, width: {u['rectWidth']}px] > viewport ({w}px)")
                            if u['textSnippet']:
                                err_msg.append(f"      Text: \"{u['textSnippet']}\"")
                    failed_checks.append("\n".join(err_msg))
                
                print(f"  [{status_str:30s}] Viewport {name} ({w}x{h})")
                
                # Check 2: Core Semantic DOM Sections
                elements_present = page.evaluate('''() => {
                    const required = [
                        '.site-header',
                        '.hero-panel',
                        '.hero-title',
                        '#overview',
                        '#topology',
                        '#directory',
                        '#principles',
                        '#design-system',
                        '#getting-started',
                        '.site-footer',
                        '#eco-drawer'
                    ];
                    return required.every(sel => document.querySelector(sel) !== null);
                }''')
                total_checks += 1
                if elements_present:
                    passed_checks += 1
                else:
                    failed_checks.append(f"{name}: Missing required DOM structural sections")

                # Check 3: Console Exceptions
                total_checks += 1
                if len(console_errors) == 0:
                    passed_checks += 1
                else:
                    failed_checks.append(f"{name}: Console errors detected: {console_errors}")

            except Exception as e:
                failed_checks.append(f"{name}: Exception during page evaluation: {e}")
            finally:
                page.close()

        # Check 4: Interactive States (Drawer lifecycle, ARIA, Theme, Copy fallback)
        print("\n[*] Running interactive state, touch, and accessibility checks...")
        context = browser.new_context(viewport={'width': 375, 'height': 667}, permissions=['clipboard-read', 'clipboard-write'])
        test_page = context.new_page()
        test_page.goto(base_url, wait_until='networkidle')
        
        # Test Theme Toggle
        total_checks += 1
        theme_btn = test_page.query_selector('#theme-toggle')
        if theme_btn:
            theme_btn.click()
            is_dark = test_page.evaluate('document.documentElement.getAttribute("data-theme") === "dark"')
            if is_dark:
                passed_checks += 1
                print("  [PASS                          ] Theme toggle activates dark mode")
            else:
                failed_checks.append("Theme toggle did not set data-theme='dark'")
        else:
            failed_checks.append("Theme toggle button not found")

        # Test Ecosystem Drawer Open/Close & ARIA sync
        total_checks += 1
        drawer_btn = test_page.query_selector('#eco-menu-toggle')
        drawer = test_page.query_selector('#eco-drawer')
        drawer_close = test_page.query_selector('#eco-drawer-close')
        if drawer_btn and drawer and drawer_close:
            drawer_btn.click()
            test_page.wait_for_timeout(100)
            is_active = test_page.evaluate('document.getElementById("eco-drawer").classList.contains("active")')
            aria_hidden = test_page.evaluate('document.getElementById("eco-drawer").getAttribute("aria-hidden")')
            aria_expanded = test_page.evaluate('document.getElementById("eco-menu-toggle").getAttribute("aria-expanded")')
            
            drawer_close.click()
            test_page.wait_for_timeout(100)
            is_closed = test_page.evaluate('!document.getElementById("eco-drawer").classList.contains("active")')
            aria_hidden_closed = test_page.evaluate('document.getElementById("eco-drawer").getAttribute("aria-hidden")')
            aria_expanded_closed = test_page.evaluate('document.getElementById("eco-menu-toggle").getAttribute("aria-expanded")')
            
            if is_active and aria_hidden == 'false' and aria_expanded == 'true' and is_closed and aria_hidden_closed == 'true' and aria_expanded_closed == 'false':
                passed_checks += 1
                print("  [PASS                          ] Ecosystem drawer open/close synchronizes visibility and ARIA attributes")
            else:
                failed_checks.append(f"Drawer ARIA state mismatch: active={is_active}, closed={is_closed}, aria_open={aria_hidden}, aria_closed={aria_hidden_closed}")
        else:
            failed_checks.append("Ecosystem drawer controls not found")

        # Test CLI Copy Button Feedback
        total_checks += 1
        copy_btn = test_page.query_selector('.btn-copy')
        if copy_btn:
            copy_btn.click()
            test_page.wait_for_timeout(200)
            btn_text = test_page.evaluate('() => (document.querySelector(".btn-copy").textContent || "").trim()')
            if '[COPIED!]' in btn_text:
                passed_checks += 1
                print("  [PASS                          ] CLI code copy button triggers visual [COPIED!] feedback")
            else:
                failed_checks.append(f"Copy button text was '{btn_text}', expected '[COPIED!]'")
        else:
            failed_checks.append("Copy button not found")

        test_page.close()
        context.close()
        browser.close()

    return passed_checks, total_checks, failed_checks

def run_ecosystem_smoke_tests(verbose=False):
    print("\n[*] Running cross-ecosystem public deployment smoke checks...")
    total_checks = 0
    passed_checks = 0
    failed_checks = []

    test_viewports = [
        (375, 667, 'mobile_375px'),
        (1280, 800, 'desktop_1280px')
    ]

    with sync_playwright() as p:
        browser = p.chromium.launch()
        
        for name, url in ECOSYSTEM_SITES.items():
            print(f"\n  Checking ecosystem node: {name} ({url})")
            for w, h, vp_name in test_viewports:
                total_checks += 1
                page = browser.new_page(viewport={'width': w, 'height': h})
                console_errors = []
                page.on('console', lambda msg: console_errors.append(msg.text) if msg.type == 'error' else None)
                
                try:
                    response = page.goto(url, wait_until='networkidle', timeout=15000)
                    status_code = response.status if response else 0
                    
                    if status_code != 200:
                        failed_checks.append(f"{name} ({vp_name}): HTTP {status_code} on {url}")
                        print(f"    [{'FAIL':4s}] {vp_name} -> HTTP {status_code}")
                        continue
                    
                    diag = page.evaluate(ELEMENT_OVERFLOW_DIAGNOSTIC_JS)
                    has_overflow = diag['hasDocOverflow'] or len(diag['uncontainedOverflows']) > 0
                    
                    if not has_overflow and len(console_errors) == 0:
                        passed_checks += 1
                        print(f"    [{'PASS':4s}] {vp_name} -> 200 OK, 0 overflow (SW={diag['scrollWidth']}, CW={diag['clientWidth']}), 0 errors")
                    else:
                        status_desc = []
                        if diag['hasDocOverflow']:
                            status_desc.append(f"DocSW={diag['scrollWidth']} > CW={diag['clientWidth']}")
                        if diag['uncontainedOverflows']:
                            status_desc.append(f"{len(diag['uncontainedOverflows'])} uncontained elements")
                        if console_errors:
                            status_desc.append(f"{len(console_errors)} console errors")
                        failed_checks.append(f"{name} ({vp_name}): {', '.join(status_desc)}")
                        print(f"    [{'FAIL':4s}] {vp_name} -> {', '.join(status_desc)}")

                except Exception as e:
                    failed_checks.append(f"{name} ({vp_name}): Network/Evaluation error: {e}")
                    print(f"    [{'FAIL':4s}] {vp_name} -> Error: {e}")
                finally:
                    page.close()
                    
        browser.close()

    return passed_checks, total_checks, failed_checks

def main():
    parser = argparse.ArgumentParser(description="Howl Responsive Regression & Continuous QA Suite")
    parser.add_argument('--ecosystem', action='store_true', help='Run cross-ecosystem public smoke tests')
    parser.add_argument('--all', action='store_true', help='Run both local Hub and ecosystem smoke tests')
    parser.add_argument('--port', type=int, default=8765, help='Local server port (default: 8765)')
    parser.add_argument('--verbose', action='store_true', help='Verbose diagnostic reporting')
    args = parser.parse_args()

    overall_passed = 0
    overall_total = 0
    all_failures = []

    # Local tests
    if not args.ecosystem or args.all:
        server = start_server(args.port)
        base_url = f'http://127.0.0.1:{args.port}/'
        try:
            p, t, f = run_local_hub_tests(base_url, verbose=args.verbose)
            overall_passed += p
            overall_total += t
            all_failures.extend(f)
        finally:
            server.shutdown()

    # Ecosystem tests
    if args.ecosystem or args.all:
        p, t, f = run_ecosystem_smoke_tests(verbose=args.verbose)
        overall_passed += p
        overall_total += t
        all_failures.extend(f)

    print(f"\n========================================================")
    print(f"FINAL QA RESULTS: {overall_passed}/{overall_total} checks passed")
    print(f"========================================================")

    if all_failures:
        print("\nFailures Encountered:")
        for fail in all_failures:
            print(f"\n{fail}")
        return 1
    else:
        print("\nAll responsive layout, element containment, and interaction tests PASSED!")
        return 0

if __name__ == '__main__':
    sys.exit(main())
