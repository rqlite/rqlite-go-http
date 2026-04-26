from __future__ import annotations

import argparse
import json
import zipfile
from io import BytesIO
from pathlib import Path

from PIL import Image


def compress_png(data: bytes, max_dim: int, colors: int) -> bytes:
    with Image.open(BytesIO(data)) as img:
        img.load()
        if img.mode not in ("RGB", "RGBA", "P", "L"):
            img = img.convert("RGBA" if "A" in img.mode else "RGB")

        if max(img.size) > max_dim:
            img.thumbnail((max_dim, max_dim), Image.Resampling.LANCZOS)

        alpha = "A" in img.getbands()
        if alpha:
            img = img.convert("RGBA").quantize(colors=min(colors, 256), method=Image.Quantize.FASTOCTREE)
        else:
            img = img.convert("P", palette=Image.Palette.ADAPTIVE, colors=min(colors, 256))

        buf = BytesIO()
        img.save(buf, format="PNG", optimize=True, compress_level=9)
        return buf.getvalue()


def compress_jpeg(data: bytes, max_dim: int, quality: int) -> bytes:
    with Image.open(BytesIO(data)) as img:
        img.load()
        if img.mode not in ("RGB", "L"):
            img = img.convert("RGB")

        if max(img.size) > max_dim:
            img.thumbnail((max_dim, max_dim), Image.Resampling.LANCZOS)

        buf = BytesIO()
        img.save(buf, format="JPEG", quality=quality, optimize=True, progressive=True)
        return buf.getvalue()


def process_docx(src: Path, dest: Path, max_dim: int, png_colors: int, jpeg_quality: int) -> dict:
    report: dict[str, object] = {
        "source": str(src),
        "dest": str(dest),
        "files_changed": 0,
        "bytes_before": src.stat().st_size,
        "media_changes": [],
    }

    with zipfile.ZipFile(src, "r") as zin, zipfile.ZipFile(dest, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as zout:
        for item in zin.infolist():
            data = zin.read(item.filename)
            new_data = data

            if item.filename.startswith("word/media/"):
                lower = item.filename.lower()
                try:
                    if lower.endswith(".png"):
                        candidate = compress_png(data, max_dim=max_dim, colors=png_colors)
                        if len(candidate) < len(data):
                            new_data = candidate
                    elif lower.endswith(".jpg") or lower.endswith(".jpeg"):
                        candidate = compress_jpeg(data, max_dim=max_dim, quality=jpeg_quality)
                        if len(candidate) < len(data):
                            new_data = candidate
                except Exception:
                    new_data = data

                if new_data is not data:
                    report["files_changed"] = int(report["files_changed"]) + 1
                    report["media_changes"].append(
                        {
                            "file": item.filename,
                            "before": len(data),
                            "after": len(new_data),
                            "saved": len(data) - len(new_data),
                        }
                    )

            zout.writestr(item, new_data)

    report["bytes_after"] = dest.stat().st_size
    report["saved_total"] = int(report["bytes_before"]) - int(report["bytes_after"])
    return report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--src", required=True)
    parser.add_argument("--dest", required=True)
    parser.add_argument("--report", required=True)
    parser.add_argument("--max-dim", type=int, default=1600)
    parser.add_argument("--png-colors", type=int, default=160)
    parser.add_argument("--jpeg-quality", type=int, default=78)
    args = parser.parse_args()

    src = Path(args.src)
    dest = Path(args.dest)
    report_path = Path(args.report)
    dest.parent.mkdir(parents=True, exist_ok=True)
    report_path.parent.mkdir(parents=True, exist_ok=True)

    report = process_docx(
        src=src,
        dest=dest,
        max_dim=args.max_dim,
        png_colors=args.png_colors,
        jpeg_quality=args.jpeg_quality,
    )
    report_path.write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(json.dumps({k: report[k] for k in ("bytes_before", "bytes_after", "saved_total", "files_changed")}))


if __name__ == "__main__":
    main()
