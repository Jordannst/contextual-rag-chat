import os
import sys
import io
import json

import fitz  # pymupdf
import google.generativeai as genai
import pytesseract
from PIL import Image


# Konfigurasi Path Tesseract untuk Windows
if os.name == "nt":
    # Coba beberapa lokasi umum instalasi Tesseract (sesuaikan jika berbeda)
    possible_paths = [
        r"C:\Program Files\Tesseract-OCR\tesseract.exe",
        r"C:\Program Files (x86)\Tesseract-OCR\tesseract.exe",
        r"C:\Users\{}\AppData\Local\Tesseract-OCR\tesseract.exe".format(os.getenv("USERNAME", "")),
        r"C:\Tesseract-OCR\tesseract.exe",
    ]

    tesseract_found = False
    for path in possible_paths:
        if os.path.exists(path):
            pytesseract.pytesseract.tesseract_cmd = path
            tesseract_found = True
            sys.stderr.write(f"[OCR] Menggunakan Tesseract di: {path}\n")
            break

    if not tesseract_found:
        # Fallback: coba gunakan Tesseract dari PATH environment variable
        try:
            _ = pytesseract.get_tesseract_version()
            tesseract_found = True
            sys.stderr.write("[OCR] Menggunakan Tesseract dari PATH environment variable.\n")
        except Exception as e:
            sys.stderr.write(
                f"[WARNING] Tesseract OCR tidak ditemukan! Pastikan sudah diinstall dan path sudah benar.\n"
            )
            sys.stderr.write(f"[WARNING] Error saat cek Tesseract dari PATH: {e}\n")
            sys.stderr.write(
                "[WARNING] OCR akan dilewati untuk halaman yang memerlukan OCR.\n"
            )


# Fungsi untuk mendapatkan bahasa OCR yang tersedia
def get_available_ocr_lang():
    """Mendeteksi bahasa yang tersedia dan return string lang yang sesuai."""
    try:
        available_langs = pytesseract.get_languages()
        sys.stderr.write(f"[OCR] Bahasa tersedia: {', '.join(available_langs)}\n")
        
        # Cek apakah eng+ind tersedia
        if "eng" in available_langs and "ind" in available_langs:
            return "eng+ind"
        elif "eng" in available_langs:
            sys.stderr.write(
                "[OCR] Bahasa Indonesia (ind) tidak tersedia, menggunakan Inggris (eng) saja.\n"
            )
            return "eng"
        else:
            sys.stderr.write(
                "[WARNING] Bahasa Inggris (eng) tidak tersedia! OCR mungkin tidak berfungsi dengan baik.\n"
            )
            # Fallback ke bahasa pertama yang tersedia
            if available_langs:
                return available_langs[0]
            return "eng"  # Default fallback
    except Exception as e:
        sys.stderr.write(f"[WARNING] Gagal mendapatkan daftar bahasa: {e}\n")
        sys.stderr.write("[OCR] Menggunakan default: eng\n")
        return "eng"  # Default fallback


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

    for page_num in range(len(doc)):
        page = doc[page_num]

        # Teks halaman biasa
        page_text = page.get_text("text") or ""
        page_text = page_text.strip()

        # Jika teks sangat sedikit, kemungkinan ini halaman hasil scan → pakai OCR
        if len(page_text.strip()) < 50:
            sys.stderr.write(f"[OCR] Halaman {page_num + 1} kosong/minim teks. Mencoba OCR...\n")
            try:
                # Cek apakah Tesseract tersedia dengan mencoba get version
                try:
                    pytesseract.get_tesseract_version()
                except Exception as tesseract_check_error:
                    sys.stderr.write(
                        f"[OCR] Tesseract tidak tersedia atau tidak bisa diakses: {tesseract_check_error}\n"
                    )
                    sys.stderr.write(
                        f"[OCR] Melewatkan OCR untuk halaman {page_num + 1}. Pastikan Tesseract sudah terinstall.\n"
                    )
                    # Lanjutkan tanpa OCR
                else:
                    # Render halaman menjadi gambar (pixmap) dengan Matrix Scaling 3x
                    # Ini meningkatkan resolusi dari default 72 DPI menjadi ~216 DPI
                    # untuk meningkatkan akurasi OCR pada teks kecil
                    matrix = fitz.Matrix(3, 3)  # Zoom 3x (3x3 = 9x lebih besar)
                    pix = page.get_pixmap(matrix=matrix)
                    img_bytes = pix.tobytes("png")

                    # Konversi ke PIL Image untuk OCR
                    img_stream = io.BytesIO(img_bytes)
                    pil_image = Image.open(img_stream)
                    
                    # Konversi ke Grayscale (mode 'L') - PENTING untuk OCR
                    # Tesseract bekerja lebih baik pada gambar hitam-putih
                    # karena mengurangi noise warna dan meningkatkan kontras
                    pil_image = pil_image.convert('L')
                    
                    # Binarization (Thresholding): Konversi ke hitam-putih murni (Binary)
                    # Threshold 150: pixel < 150 jadi hitam pekat (0), >= 150 jadi putih bersih (255)
                    # Ini membuat teks lebih tajam dan meningkatkan akurasi OCR
                    pil_image = pil_image.point(lambda x: 0 if x < 150 else 255, '1')
                    
                    sys.stderr.write(
                        f"[OCR] Gambar diproses: {pil_image.width}x{pil_image.height}px, mode={pil_image.mode}\n"
                    )

                    # Dapatkan bahasa OCR yang tersedia
                    ocr_lang = get_available_ocr_lang()
                    
                    # Konfigurasi Tesseract untuk dokumen tabular/struk
                    # PSM 6: Assume a single uniform block of text
                    # Memaksa Tesseract membaca baris demi baris dari kiri ke kanan
                    # sehingga harga di kolom kanan tidak terlewat
                    custom_config = r'--oem 3 --psm 6'
                    
                    # Jalankan OCR dengan bahasa dan konfigurasi khusus
                    ocr_text = pytesseract.image_to_string(pil_image, lang=ocr_lang, config=custom_config).strip()
                    sys.stderr.write(
                        f"[OCR] Berhasil: {len(ocr_text)} karakter pada halaman {page_num + 1}.\n"
                    )

                    if ocr_text:
                        # Gabungkan hasil OCR ke teks halaman (dengan label agar jelas di RAG)
                        if page_text:
                            page_text = page_text + "\n[OCR RESULT]\n" + ocr_text
                        else:
                            page_text = "[OCR RESULT]\n" + ocr_text
                    else:
                        sys.stderr.write(
                            f"[OCR] Tidak ada teks yang terdeteksi pada halaman {page_num + 1}.\n"
                        )
            except Exception as e:
                sys.stderr.write(f"[pdf_processor] OCR gagal pada halaman {page_num + 1}: {e}\n")
                import traceback
                sys.stderr.write(f"[pdf_processor] Traceback: {traceback.format_exc()}\n")

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
            header = f"=== HALAMAN {page_num + 1} ==="
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


