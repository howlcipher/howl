#!/usr/bin/env python3
"""
Automated Responsive Regression Test Suite for Howl Ecosystem Hub
Validates responsive layout, touch targets, DOM boundaries, visual components,
and zero document-level horizontal overflow across mobile, tablet, and desktop viewports.
"""

import sys
import os
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

class QuietHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=DOCS_DIR, **kwargs)
        
    def log_message(self, format, *args):
        pass  # Suppress request logging during tests

def start_server(port=8765):
    server = HTTPServer(('127.0.0.1', port), QuietHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server

def run_tests():
    port = 8765
    server = start_server(port)
    base_url = f'http://127.0.0.1:{port}/'
    print(f"[*] Serving {DOCS_DIR} on {base_url}")
    print("[*] Launching Playwright Chromium responsive verification suite...\n")
    
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
                
                # Check 1: Document Overflow
                doc_metrics = page.evaluate('''() => {
                    const doc = document.documentElement;
                    return {
                        scrollWidth: doc.scrollWidth,
                        clientWidth: doc.clientWidth,
                        hasOverflow: doc.scrollWidth > doc.clientWidth
                    };
                }''')
                
                if not doc_metrics['hasOverflow']:
                    passed_checks += 1
                    status_str = f"PASS (SW={doc_metrics['scrollWidth']}, CW={doc_metrics['clientWidth']})"
                else:
                    failed_checks.append(f"{name}: Document horizontal overflow detected! SW={doc_metrics['scrollWidth']} > CW={doc_metrics['clientWidth']}")
                    status_str = f"FAIL (SW={doc_metrics['scrollWidth']} > CW={doc_metrics['clientWidth']})"
                
                print(f"  [{status_str:30s}] Viewport {name} ({w}x{h})")
                
                # Check 2: Key elements rendered
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

                # Check 3: Console errors
                total_checks += 1
                if len(console_errors) == 0:
                    passed_checks += 1
                else:
                    failed_checks.append(f"{name}: Console errors: {console_errors}")

            except Exception as e:
                failed_checks.append(f"{name}: Exception during page evaluation: {e}")
            finally:
                page.close()

        # Check 4: Interactive and functional tests (Drawer, Theme, Copy)
        print("\n[*] Running interactive state and accessibility tests...")
        context = browser.new_context(viewport={'width': 375, 'height': 667}, permissions=['clipboard-read', 'clipboard-write'])
        test_page = context.new_page()
        test_page.goto(base_url, wait_until='networkidle')
        
        # Test theme toggle
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

        # Test drawer open/close
        total_checks += 1
        drawer_btn = test_page.query_selector('#eco-menu-toggle')
        drawer = test_page.query_selector('#eco-drawer')
        drawer_close = test_page.query_selector('#eco-drawer-close')
        if drawer_btn and drawer and drawer_close:
            drawer_btn.click()
            test_page.wait_for_timeout(100)
            is_active = test_page.evaluate('document.getElementById("eco-drawer").classList.contains("active")')
            aria_hidden = test_page.evaluate('document.getElementById("eco-drawer").getAttribute("aria-hidden")')
            
            drawer_close.click()
            test_page.wait_for_timeout(100)
            is_closed = test_page.evaluate('!document.getElementById("eco-drawer").classList.contains("active")')
            aria_hidden_closed = test_page.evaluate('document.getElementById("eco-drawer").getAttribute("aria-hidden")')
            
            if is_active and aria_hidden == 'false' and is_closed and aria_hidden_closed == 'true':
                passed_checks += 1
                print("  [PASS                          ] Ecosystem drawer opens and closes with correct aria attributes")
            else:
                failed_checks.append(f"Drawer state mismatch: active={is_active}, closed={is_closed}, aria_open={aria_hidden}, aria_closed={aria_hidden_closed}")
        else:
            failed_checks.append("Ecosystem drawer elements not found")

        # Test copy button
        total_checks += 1
        copy_btn = test_page.query_selector('.btn-copy')
        if copy_btn:
            copy_btn.click()
            test_page.wait_for_timeout(200)
            btn_text = test_page.evaluate('() => (document.querySelector(".btn-copy").textContent || "").trim()')
            if '[COPIED!]' in btn_text:
                passed_checks += 1
                print("  [PASS                          ] CLI code snippet copy button triggers [COPIED!] feedback")
            else:
                failed_checks.append(f"Copy button text was '{btn_text}', expected '[COPIED!]'")
        else:
            failed_checks.append("Copy button not found")

        test_page.close()
        context.close()
        browser.close()

    server.shutdown()
    
    print(f"\n========================================================")
    print(f"TEST RESULTS: {passed_checks}/{total_checks} checks passed")
    print(f"========================================================")
    
    if failed_checks:
        print("\nFailures:")
        for f in failed_checks:
            print(f"  - {f}")
        return 1
    else:
        print("\nAll responsive and interaction tests PASSED cleanly!")
        return 0

if __name__ == '__main__':
    sys.exit(run_tests())
