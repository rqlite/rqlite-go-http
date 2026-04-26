from __future__ import annotations

import json
from pathlib import Path

from pypdf import PdfReader

try:
    import fitz  # PyMuPDF
except Exception:  # pragma: no cover
    fitz = None


PDF_PATH = Path(r"C:\Users\venka\Downloads\Rhan.pdf")
OUT_DIR = Path(r"C:\opensource\rqlite\rqlite-go-http\tmp\rhan_pdf")
TEXT_JSON = OUT_DIR / "pdf_text.json"
IMAGES_DIR = OUT_DIR / "images"
PAGES_DIR = OUT_DIR / "pages"


def extract_text() -> list[dict]:
    reader = PdfReader(str(PDF_PATH))
    pages = []
    for idx, page in enumerate(reader.pages, start=1):
        text = (page.extract_text() or "").replace("\x00", " ").strip()
        pages.append(
            {
                "page": idx,
                "text": text,
            }
        )
    return pages


def extract_visuals() -> list[dict]:
    visuals: list[dict] = []
    if fitz is None:
        return visuals

    doc = fitz.open(str(PDF_PATH))
    for idx in range(doc.page_count):
        page = doc.load_page(idx)
        page_path = PAGES_DIR / f"page-{idx + 1:02d}.png"
        pix = page.get_pixmap(matrix=fitz.Matrix(2, 2), alpha=False)
        pix.save(str(page_path))

        image_entries = []
        for image_num, img in enumerate(page.get_images(full=True), start=1):
            xref = img[0]
            base = doc.extract_image(xref)
            ext = base.get("ext", "png")
            image_path = IMAGES_DIR / f"page-{idx + 1:02d}-img-{image_num:02d}.{ext}"
            image_path.write_bytes(base["image"])
            image_entries.append(
                {
                    "image_index": image_num,
                    "path": str(image_path),
                    "ext": ext,
                    "width": base.get("width"),
                    "height": base.get("height"),
                    "bytes": len(base["image"]),
                }
            )

        visuals.append(
            {
                "page": idx + 1,
                "page_image": str(page_path),
                "images": image_entries,
            }
        )
    return visuals


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    IMAGES_DIR.mkdir(parents=True, exist_ok=True)
    PAGES_DIR.mkdir(parents=True, exist_ok=True)

    text_pages = extract_text()
    visuals = extract_visuals()
    payload = {
        "pdf": str(PDF_PATH),
        "page_count": len(text_pages),
        "pages": text_pages,
        "visuals": visuals,
        "has_pymupdf": fitz is not None,
    }
    TEXT_JSON.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    print(str(TEXT_JSON))


if __name__ == "__main__":
    main()
