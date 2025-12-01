import os
import sys
import io
import json

import fitz  # pymupdf
import google.generativeai as genai


PROMPT = (
    "Deskripsikan gambar atau grafik ini secara detail untuk keperluan pencarian data. "
    "Fokus pada isi visual, teks di dalam gambar (jika ada), hubungan antar elemen, "
    "dan konteks yang mungkin relevan untuk penelusuran informasi."
)


def configure_gemini():
    api_key = os.environ.get("GEMINI_API_KEY")
    if not api_key:
        # Fallback: if multiple keys are used in a comma-separated env
        multi = os.environ.get("GEMINI_API_KEYS")
        if multi:
            api_key = multi.split(",")[0].strip()

    if not api_key:
        raise RuntimeError(
            "GEMINI_API_KEY (atau GEMINI_API_KEYS) tidak ditemukan di environment."
        )

    genai.configure(api_key=api_key)
    # Untuk kompatibilitas dengan library yang masih v1beta,
    # gunakan gemini-pro-vision yang umum tersedia.
    return genai.GenerativeModel("gemini-pro-vision")


def describe_image_bytes(model, image_bytes: bytes) -> str:
    try:
        resp = model.generate_content(
            [
                PROMPT,
                {
                    "mime_type": "image/png",
                    "data": image_bytes,
                },
            ]
        )
        return (resp.text or "").strip()
    except Exception as e:
        # Jangan gagal total hanya karena satu gambar
        sys.stderr.write(f"[pdf_processor] Gagal mendeskripsikan gambar: {e}\n")
        return ""


def process_pdf(path: str) -> str:
    model = configure_gemini()

    doc = fitz.open(path)
    result_pages = []

    for page_index in range(len(doc)):
        page = doc[page_index]

        # Teks halaman
        page_text = page.get_text("text") or ""
        page_text = page_text.strip()

        # Gambar pada halaman
        image_descriptions = []
        images = page.get_images(full=True)
        seen_xrefs = set()

        for img in images:
            xref = img[0]
            if xref in seen_xrefs:
                continue
            seen_xrefs.add(xref)

            pix = fitz.Pixmap(doc, xref)
            try:
                if pix.n > 4:  # convert CMYK / other to RGB
                    pix = fitz.Pixmap(fitz.csRGB, pix)

                # Ambil bytes PNG dari pixmap (PyMuPDF >= 1.18)
                image_bytes = pix.tobytes("png")

                desc = describe_image_bytes(model, image_bytes)
                if desc:
                    image_descriptions.append(desc)
            finally:
                pix = None

        combined_parts = []
        if page_text:
            combined_parts.append(page_text)
        if image_descriptions:
            combined_parts.append(
                "DESKRIPSI GAMBAR:\n" + "\n\n".join(image_descriptions)
            )

        if combined_parts:
            header = f"=== HALAMAN {page_index + 1} ==="
            result_pages.append(header + "\n" + "\n\n".join(combined_parts))

    doc.close()
    return "\n\n".join(result_pages).strip()


def main():
    if len(sys.argv) < 2:
        sys.stderr.write("Usage: python pdf_processor.py <pdf_path>\n")
        sys.exit(1)

    pdf_path = sys.argv[1]
    if not os.path.exists(pdf_path):
        sys.stderr.write(f"File not found: {pdf_path}\n")
        sys.exit(1)

    try:
        content = process_pdf(pdf_path)
    except Exception as e:
        # Laporkan error ke stderr dan keluar dengan non-zero agar Go bisa mendeteksi
        sys.stderr.write(json.dumps({"error": str(e)}) + "\n")
        sys.exit(1)

    # Hasil akhir dikirim ke stdout agar bisa dibaca oleh proses Go
    if content:
        # Paksa encode ke UTF-8 agar aman di Windows (cp1252)
        sys.stdout.buffer.write(content.encode("utf-8", errors="ignore"))


if __name__ == "__main__":
    main()


