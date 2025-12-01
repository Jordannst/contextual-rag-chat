import os
import sys
import json

import pandas as pd


def usage_and_exit():
    sys.stderr.write("Usage: python data_processor.py <path_to_csv_or_excel>\n")
    sys.exit(1)


def read_tabular_file(path: str) -> pd.DataFrame:
    ext = os.path.splitext(path)[1].lower()

    if ext == ".csv":
        df = pd.read_csv(path)
    elif ext in [".xlsx", ".xls"]:
        # pandas akan memilih engine yang sesuai (openpyxl/xlrd) jika sudah terinstall
        df = pd.read_excel(path)
    else:
        raise ValueError(f"Unsupported file type for data_processor: {ext}")

    # Bersihkan data: ganti NaN dengan string kosong
    df = df.fillna("")
    return df


def dataframe_to_narrative(df: pd.DataFrame) -> str:
    """
    Ubah setiap baris DataFrame menjadi kalimat deskriptif.
    Contoh:
    "Baris 1: Kolom Nama=Budi, Kolom Gaji=5000000, Kolom Divisi=IT."
    """
    lines: list[str] = []

    for idx, row in df.iterrows():
        parts = []
        for col in df.columns:
            value = row[col]
            # Konversi ke string dan strip whitespace
            text_value = str(value).strip()
            parts.append(f"Kolom {col}={text_value}")

        line = f"Baris {idx + 1}: " + ", ".join(parts) + "."
        lines.append(line)

    return "\n".join(lines)


def process_file(path: str) -> str:
    df = read_tabular_file(path)
    return dataframe_to_narrative(df)


def main():
    if len(sys.argv) < 2:
        usage_and_exit()

    file_path = sys.argv[1]
    if not os.path.exists(file_path):
        sys.stderr.write(f"File not found: {file_path}\n")
        sys.exit(1)

    try:
        content = process_file(file_path)
    except Exception as e:
        # Laporkan error ke stderr dalam bentuk JSON agar mudah di-debug dari Go
        sys.stderr.write(json.dumps({"error": str(e)}) + "\n")
        sys.exit(1)

    if content:
        # Tulis hasil ke stdout sebagai UTF-8 (aman di Windows, sama seperti pdf_processor)
        sys.stdout.buffer.write(content.encode("utf-8", errors="ignore"))


if __name__ == "__main__":
    main()


