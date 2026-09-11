"""Rendered local or live responsive checks; screenshots remain local."""

import argparse
from pathlib import Path

from playwright.sync_api import sync_playwright


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url")
    args = parser.parse_args()
    url = args.url or (Path(__file__).resolve().parents[1] / "docs/ecosystem.html").as_uri()
    screenshots = Path("/tmp/howl_ecosystem_screenshots")
    screenshots.mkdir(exist_ok=True)
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        for width in (375, 768, 1440):
            page = browser.new_page(viewport={"width": width, "height": 1000})
            page.goto(url, wait_until="networkidle")
            assert page.locator("h1").count() == 1
            assert page.title()
            assert page.locator('meta[name="description"]').get_attribute("content")
            assert page.locator("main").is_visible()
            assert page.evaluate("document.documentElement.scrollWidth <= innerWidth"), width
            for link in page.locator('a[href^="#"]').all():
                fragment = link.get_attribute("href")[1:]
                assert page.locator(f'[id="{fragment}"]').count() == 1, fragment
            page.keyboard.press("Tab")
            assert page.locator(".skip-link").evaluate("e => e === document.activeElement")
            page.screenshot(path=str(screenshots / f"site_{width}.png"), full_page=True)
            print(f"PASS {width}px: title, landmarks, navigation, keyboard, no overflow")
            page.close()
        browser.close()


if __name__ == "__main__":
    main()
